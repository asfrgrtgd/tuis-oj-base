package core

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// NewRouter constructs the Gin engine with routes wired.
func NewRouter(cfg Config, store *sessions.CookieStore, authService AuthService, db *pgxpool.Pool, redisClient *redis.Client) *gin.Engine {
	startedAt := time.Now()
	r := gin.Default()

	// Global middleware: origin/CORS -> session -> CSRF
	r.Use(OriginRefererMiddleware(cfg))
	r.Use(SessionMiddleware(cfg, store))
	r.Use(CSRFMiddleware(cfg, store))

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	userRepo := NewPgUserRepository(db)
	problemRepo := NewPgProblemRepository(db)
	subRepo := NewPgSubmissionRepository(db)
	queue := NewRedisQueue(redisClient)
	metricsService := NewMetricsService(redisClient)
	noticeRepo := NewPgNoticeRepository(db)
	api := r.Group("/api/v1")
	{
		api.POST("/auth/login", func(c *gin.Context) {
			var req struct {
				UserID   string `json:"userid"`
				Password string `json:"password"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json")
				return
			}

			user, err := authService.Authenticate(req.UserID, req.Password)
			if err != nil {
				respondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "ユーザーIDまたはパスワードが違います。")
				return
			}

			session, err := store.Get(c.Request, sessionName)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "session error")
				return
			}

			// reset session values (simple rotation)
			session.Values = map[interface{}]interface{}{}
			session.Values["userid"] = user.Username
			session.Values["role"] = user.Role
			applySessionOptions(cfg, session)

			if err := session.Save(c.Request, c.Writer); err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to set session")
				return
			}

			c.JSON(http.StatusOK, gin.H{"user": gin.H{"userid": user.Username, "role": user.Role}})
		})

		api.POST("/auth/logout", func(c *gin.Context) {
			sessionAny, _ := c.Get("session")
			sess, _ := sessionAny.(*sessions.Session)
			if sess == nil {
				respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "ログインが必要です。")
				return
			}
			sess.Values = map[interface{}]interface{}{}
			applySessionOptions(cfg, sess)
			sess.Options.MaxAge = -1 // Must be set AFTER applySessionOptions to properly delete cookie
			if err := sess.Save(c.Request, c.Writer); err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to clear session")
				return
			}
			c.Status(http.StatusNoContent)
		})

		api.GET("/users/me", func(c *gin.Context) {
			sessionAny, _ := c.Get("session")
			sess, _ := sessionAny.(*sessions.Session)
			userid, _ := sess.Values["userid"].(string)
			if strings.TrimSpace(userid) == "" {
				respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "ログインが必要です。")
				return
			}

			ctx := c.Request.Context()
			u, err := userRepo.FindByUsername(ctx, userid)
			if err != nil {
				respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "ユーザーが存在しません")
				return
			}
			subCount, err := subRepo.CountByUser(ctx, u.ID)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to count submissions")
				return
			}
			solvedCount, err := subRepo.CountSolvedProblemsByUser(ctx, u.ID)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to count solved problems")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"userid":           u.Username,
				"role":             u.Role,
				"solved_count":     solvedCount,
				"submission_count": subCount,
				"created_at":       u.CreatedAt,
			})
		})

		api.GET("/users/:userid", func(c *gin.Context) {
			if _, ok := requireLogin(c); !ok {
				return
			}

			uid := c.Param("userid")
			ctx := c.Request.Context()
			u, err := userRepo.FindByUsername(ctx, uid)
			if err != nil {
				respondError(c, http.StatusNotFound, "NOT_FOUND", "ユーザーが見つかりません")
				return
			}
			subCount, err := subRepo.CountByUser(ctx, u.ID)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to count submissions")
				return
			}
			solvedCount, err := subRepo.CountSolvedProblemsByUser(ctx, u.ID)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to count solved problems")
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"userid":           u.Username,
				"role":             u.Role,
				"solved_count":     solvedCount,
				"submission_count": subCount,
				"created_at":       u.CreatedAt,
			})
		})

		api.POST("/submissions", func(c *gin.Context) {
			// Simple session auth
			sessionAny, _ := c.Get("session")
			sess, _ := sessionAny.(*sessions.Session)
			useridVal := sess.Values["userid"]
			username, _ := useridVal.(string)
			if strings.TrimSpace(username) == "" {
				respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "ログインが必要です。")
				return
			}

			var req struct {
				ProblemID int64  `json:"problem_id"`
				Language  string `json:"language"`
				Source    string `json:"source_code"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json")
				return
			}
			if req.ProblemID <= 0 || strings.TrimSpace(req.Language) == "" || strings.TrimSpace(req.Source) == "" {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "problem_id, language, source_code は必須です")
				return
			}

			ctx := c.Request.Context()
			user, err := userRepo.FindByUsername(ctx, username)
			if err != nil {
				respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "ユーザーが存在しません")
				return
			}

			// problem check
			isPublic, err := problemRepo.ExistsAndPublic(ctx, req.ProblemID)
			if err != nil {
				respondError(c, http.StatusNotFound, "NOT_FOUND", "問題が見つかりません")
				return
			}
			if !isPublic {
				respondError(c, http.StatusForbidden, "FORBIDDEN", "非公開の問題です")
				return
			}
			if !isSupportedLanguage(req.Language) {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "サポートされていない言語です")
				return
			}

			// Reserve ID by inserting with empty source_path first
			sourcePath := ""
			subID, createdAt, err := subRepo.Create(ctx, user.ID, req.ProblemID, req.Language, sourcePath)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to create submission")
				return
			}

			dir := filepath.Join(cfg.SubmissionDir, strconv.FormatInt(subID, 10))
			if err := ensureDir(dir); err != nil {
				_ = subRepo.Delete(ctx, subID)
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to prepare dir")
				return
			}
			srcPath := filepath.Join(dir, "source")
			if err := os.WriteFile(srcPath, []byte(req.Source), 0644); err != nil {
				_ = subRepo.Delete(ctx, subID)
				_ = os.RemoveAll(dir)
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to save source")
				return
			}

			if _, err := db.Exec(ctx, `UPDATE submissions SET source_path=$1 WHERE id=$2`, srcPath, subID); err != nil {
				_ = subRepo.Delete(ctx, subID)
				_ = os.RemoveAll(dir)
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to update source path")
				return
			}

			// enqueue
			if err := queue.Enqueue(ctx, "pending_submissions", strconv.FormatInt(subID, 10)); err != nil {
				_ = subRepo.Delete(ctx, subID)
				_ = os.RemoveAll(dir)
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to enqueue")
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"id":         subID,
				"problem_id": req.ProblemID,
				"language":   req.Language,
				"status":     "pending",
				"verdict":    nil,
				"time_ms":    nil,
				"memory_kb":  nil,
				"created_at": createdAt,
			})
		})

		api.GET("/languages", func(c *gin.Context) {
			if _, ok := requireLogin(c); !ok {
				return
			}
			c.JSON(http.StatusOK, gin.H{"languages": supportedLanguages})
		})

		// お知らせ一覧
		api.GET("/notices", func(c *gin.Context) {
			if _, ok := requireLogin(c); !ok {
				return
			}

			page, perPage, err := parsePagination(c.Query("page"), c.Query("per_page"))
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
				return
			}
			ctx := c.Request.Context()
			items, total, err := noticeRepo.List(ctx, page, perPage)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch notices")
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"items":       items,
				"page":        page,
				"per_page":    perPage,
				"total_items": total,
				"total_pages": calcTotalPages(total, perPage),
			})
		})

		api.GET("/notices/:id", func(c *gin.Context) {
			if _, ok := requireLogin(c); !ok {
				return
			}

			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil || id <= 0 {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
				return
			}
			ctx := c.Request.Context()
			n, err := noticeRepo.Get(ctx, id)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					respondError(c, http.StatusNotFound, "NOT_FOUND", "notice not found")
					return
				}
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch notice")
				return
			}
			c.JSON(http.StatusOK, n)
		})

		admin := api.Group("/admin")
		admin.Use(AdminOnly())
		metrics := admin.Group("/metrics")
		{
			metrics.GET("/overview", func(c *gin.Context) {
				ctx := c.Request.Context()
				queueMetrics, workers, err := metricsService.Overview(ctx)
				if err != nil {
					respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load metrics")
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"queues":  queueMetrics,
					"workers": workers,
				})
			})

			metrics.GET("/queues", func(c *gin.Context) {
				ctx := c.Request.Context()
				queueMetrics, err := metricsService.Queue(ctx)
				if err != nil {
					respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load queue metrics")
					return
				}
				c.JSON(http.StatusOK, queueMetrics)
			})

			metrics.GET("/workers", func(c *gin.Context) {
				ctx := c.Request.Context()
				workers, err := metricsService.Workers(ctx)
				if err != nil {
					respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load workers")
					return
				}
				c.JSON(http.StatusOK, gin.H{"workers": workers})
			})

			metrics.GET("/workers/:id", func(c *gin.Context) {
				ctx := c.Request.Context()
				id := c.Param("id")
				hb, err := metricsService.WorkerByID(ctx, id)
				if err != nil {
					if errors.Is(err, redis.Nil) {
						respondError(c, http.StatusNotFound, "NOT_FOUND", "worker not found")
						return
					}
					respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load worker")
					return
				}
				c.JSON(http.StatusOK, hb)
			})
		}
		admin.GET("/system/status", func(c *gin.Context) {
			ctx := c.Request.Context()
			st, err := CollectSystemStatus(ctx, metricsService, startedAt)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load system status")
				return
			}
			c.JSON(http.StatusOK, st)
		})

		admin.POST("/submissions/bulk_test", func(c *gin.Context) {
			var req struct {
				ProblemID  int64  `json:"problem_id"`
				Language   string `json:"language"`
				Count      int    `json:"count"`
				SourceCode string `json:"source_code"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json")
				return
			}
			if req.Count <= 0 {
				req.Count = 10
			}
			if req.Count > 100 {
				req.Count = 100
			}
			if req.ProblemID <= 0 {
				req.ProblemID = 1
			}
			if strings.TrimSpace(req.Language) == "" {
				req.Language = "c"
			}
			if !isSupportedLanguage(req.Language) {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "サポートされていない言語です")
				return
			}
			if strings.TrimSpace(req.SourceCode) == "" {
				req.SourceCode = defaultSourceFor(req.Language)
			}

			ctx := c.Request.Context()
			// use current admin user
			sessionAny, _ := c.Get("session")
			sess, _ := sessionAny.(*sessions.Session)
			username, _ := sess.Values["userid"].(string)
			user, err := userRepo.FindByUsername(ctx, username)
			if err != nil {
				respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "ユーザーが存在しません")
				return
			}

			exists, err := problemRepo.Exists(ctx, req.ProblemID)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "問題の存在確認に失敗しました")
				return
			}
			if !exists {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "problem_id が不正です")
				return
			}

			ids := make([]int64, 0, req.Count)
			for i := 0; i < req.Count; i++ {
				subID, err := createSubmissionWithSource(ctx, cfg, subRepo, db, queue, user.ID, req.ProblemID, req.Language, req.SourceCode)
				if err != nil {
					respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", fmt.Sprintf("failed at %d/%d: %v", i+1, req.Count, err))
					return
				}
				ids = append(ids, subID)
			}
			c.JSON(http.StatusCreated, gin.H{
				"created":  ids,
				"count":    len(ids),
				"problem":  req.ProblemID,
				"language": req.Language,
			})
		})

		// alias: /submissions/test -> bulk_test
		admin.POST("/submissions/test", func(c *gin.Context) {
			// forward to bulk_test handler
			c.Request.URL.Path = "/api/v1/admin/submissions/bulk_test"
			r.HandleContext(c)
		})

		// お知らせ CRUD（管理者のみ）
		admin.GET("/notices", func(c *gin.Context) {
			page, perPage, err := parsePagination(c.Query("page"), c.Query("per_page"))
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
				return
			}
			ctx := c.Request.Context()
			items, total, err := noticeRepo.List(ctx, page, perPage)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch notices")
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"items":       items,
				"page":        page,
				"per_page":    perPage,
				"total_items": total,
				"total_pages": calcTotalPages(total, perPage),
			})
		})

		admin.POST("/notices", func(c *gin.Context) {
			var req struct {
				Title string `json:"title"`
				Body  string `json:"body"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json")
				return
			}
			req.Title = strings.TrimSpace(req.Title)
			req.Body = strings.TrimSpace(req.Body)
			if req.Title == "" || req.Body == "" {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "title と body は必須です")
				return
			}
			ctx := c.Request.Context()
			n, err := noticeRepo.Create(ctx, req.Title, req.Body)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to create notice")
				return
			}
			c.JSON(http.StatusCreated, n)
		})

		admin.PATCH("/notices/:id", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil || id <= 0 {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
				return
			}
			var req struct {
				Title string `json:"title"`
				Body  string `json:"body"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json")
				return
			}
			if strings.TrimSpace(req.Title) == "" && strings.TrimSpace(req.Body) == "" {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "title か body のいずれかを指定してください")
				return
			}
			// 部分更新: 未指定は既存を維持
			ctx := c.Request.Context()
			current, err := noticeRepo.Get(ctx, id)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					respondError(c, http.StatusNotFound, "NOT_FOUND", "notice not found")
					return
				}
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch notice")
				return
			}
			title := strings.TrimSpace(req.Title)
			if title == "" {
				title = current.Title
			}
			body := strings.TrimSpace(req.Body)
			if body == "" {
				body = current.Body
			}
			n, err := noticeRepo.Update(ctx, id, title, body)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to update notice")
				return
			}
			c.JSON(http.StatusOK, n)
		})

		admin.DELETE("/notices/:id", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil || id <= 0 {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
				return
			}
			ctx := c.Request.Context()
			if err := noticeRepo.Delete(ctx, id); err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to delete notice")
				return
			}
			c.Status(http.StatusNoContent)
		})

		admin.POST("/users", func(c *gin.Context) {
			var req struct {
				UserID   string `json:"userid"`
				Password string `json:"password"`
				Role     string `json:"role"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json")
				return
			}
			req.UserID = strings.TrimSpace(req.UserID)
			req.Role = strings.TrimSpace(req.Role)
			if req.UserID == "" || req.Password == "" {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "userid and password are required")
				return
			}
			if req.Role == "" {
				req.Role = "user"
			}
			if req.Role != "user" && req.Role != "admin" {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid role")
				return
			}

			hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to hash password")
				return
			}
			ctx := c.Request.Context()
			if _, err := userRepo.Create(ctx, req.UserID, string(hash), req.Role); err != nil {
				// naive duplicate detection
				if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
					respondError(c, http.StatusConflict, "CONFLICT", "userid already exists")
					return
				}
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to create user")
				return
			}

			// created_at を含むレスポンスを返すために再取得
			record, err := userRepo.FindByUsername(ctx, req.UserID)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load created user")
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"id":         record.ID,
				"userid":     record.Username,
				"role":       record.Role,
				"created_at": record.CreatedAt,
			})
		})

		admin.GET("/users", func(c *gin.Context) {
			page, perPage, err := parsePagination(c.Query("page"), c.Query("per_page"))
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
				return
			}
			ctx := c.Request.Context()
			items, total, err := userRepo.List(ctx, page, perPage)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch users")
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"items":       items,
				"page":        page,
				"per_page":    perPage,
				"total_items": total,
				"total_pages": calcTotalPages(total, perPage),
			})
		})

		admin.GET("/problems/template", func(c *gin.Context) {
			data, err := buildProblemTemplateZip()
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to build template")
				return
			}
			c.Header("Content-Type", "application/zip")
			c.Header("Content-Disposition", "attachment; filename=two-string.zip")
			c.Data(http.StatusOK, "application/zip", data)
		})

		admin.POST("/problems/import", func(c *gin.Context) {
			fileHeader, err := c.FormFile("file")
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "file フィールドに zip を指定してください")
				return
			}
			if fileHeader.Size > maxProblemImportSize {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "ファイルが大きすぎます (8MB 以下にしてください)")
				return
			}
			file, err := fileHeader.Open()
			if err != nil {
				respondError(c, http.StatusBadRequest, "INVALID_PROBLEM_PACKAGE", "ファイルを開けません")
				return
			}
			defer file.Close()
			limited := io.LimitReader(file, maxProblemImportSize+1024)
			data, err := io.ReadAll(limited)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "アップロードの読み取りに失敗しました")
				return
			}
			if int64(len(data)) > maxProblemImportSize {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "ファイルが大きすぎます (8MB 以下にしてください)")
				return
			}

			pkg, err := ParseProblemArchive(data)
			if err != nil {
				respondError(c, http.StatusBadRequest, "INVALID_PROBLEM_PACKAGE", err.Error())
				return
			}

			ctx := c.Request.Context()
			problemID, err := problemRepo.CreateWithTestcases(ctx, pkg)
			if err != nil {
				if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
					respondError(c, http.StatusConflict, "CONFLICT", "同じ slug の問題が既に存在します")
					return
				}
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "問題の保存に失敗しました")
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"id":              problemID,
				"title":           pkg.Title,
				"slug":            pkg.Slug,
				"time_limit_ms":   pkg.TimeLimitMS,
				"memory_limit_kb": pkg.MemoryLimitKB,
				"is_public":       pkg.IsPublic,
			})
		})

		admin.GET("/problems", func(c *gin.Context) {
			page, perPage, err := parsePagination(c.Query("page"), c.Query("per_page"))
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
				return
			}
			ctx := c.Request.Context()
			items, total, err := problemRepo.AdminList(ctx, page, perPage)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch problems")
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"items":       items,
				"page":        page,
				"per_page":    perPage,
				"total_items": total,
				"total_pages": calcTotalPages(total, perPage),
			})
		})

		admin.GET("/problems/:id/download", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil || id <= 0 {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
				return
			}
			ctx := c.Request.Context()
			detail, err := problemRepo.FindDetailAdmin(ctx, id)
			if err != nil {
				respondError(c, http.StatusNotFound, "NOT_FOUND", "problem not found")
				return
			}
			cases, err := problemRepo.ListTestcases(ctx, id)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to load testcases")
				return
			}
			zipBytes, err := buildProblemZipFromDB(*detail, cases)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to build archive")
				return
			}
			c.Header("Content-Type", "application/zip")
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.zip", detail.Slug))
			c.Data(http.StatusOK, "application/zip", zipBytes)
		})

		admin.PATCH("/problems/:id", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil || id <= 0 {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
				return
			}
			var req struct {
				Title         *string  `json:"title"`
				StatementMD   *string  `json:"statement_md"`
				TimeLimitMS   *int32   `json:"time_limit_ms"`
				MemoryLimitKB *int32   `json:"memory_limit_kb"`
				IsPublic      *bool    `json:"is_public"`
				CheckerType   *string  `json:"checker_type"`
				CheckerEps    *float64 `json:"checker_eps"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid json")
				return
			}
			ctx := c.Request.Context()
			exists, err := problemRepo.Exists(ctx, id)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch problem")
				return
			}
			if !exists {
				respondError(c, http.StatusNotFound, "NOT_FOUND", "problem not found")
				return
			}
			if err := problemRepo.UpdateProblem(ctx, id, ProblemUpdateInput{
				Title:         req.Title,
				StatementMD:   req.StatementMD,
				TimeLimitMS:   req.TimeLimitMS,
				MemoryLimitKB: req.MemoryLimitKB,
				IsPublic:      req.IsPublic,
				CheckerType:   req.CheckerType,
				CheckerEps:    req.CheckerEps,
			}); err != nil {
				if strings.Contains(err.Error(), "checker") || strings.Contains(err.Error(), "limit") {
					respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
					return
				}
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to update problem")
				return
			}
			c.Status(http.StatusNoContent)
		})

		admin.GET("/problems/:id/stats", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil || id <= 0 {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
				return
			}
			ctx := c.Request.Context()
			stats, err := problemRepo.ProblemStats(ctx, id)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					respondError(c, http.StatusNotFound, "NOT_FOUND", "problem not found")
					return
				}
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch stats")
				return
			}
			c.JSON(http.StatusOK, stats)
		})

		admin.GET("/users/:userid/submissions", func(c *gin.Context) {
			page, perPage, err := parsePagination(c.Query("page"), c.Query("per_page"))
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
				return
			}
			ctx := c.Request.Context()
			user, err := userRepo.FindByUsername(ctx, c.Param("userid"))
			if err != nil {
				respondError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
				return
			}
			items, total, err := subRepo.ListByUser(ctx, user.ID, nil, page, perPage)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch submissions")
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"items":       items,
				"page":        page,
				"per_page":    perPage,
				"total_items": total,
				"total_pages": calcTotalPages(total, perPage),
			})
		})

		admin.GET("/problems/:id/submissions", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil || id <= 0 {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
				return
			}
			page, perPage, err := parsePagination(c.Query("page"), c.Query("per_page"))
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
				return
			}
			ctx := c.Request.Context()
			exists, err := problemRepo.Exists(ctx, id)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch problem")
				return
			}
			if !exists {
				respondError(c, http.StatusNotFound, "NOT_FOUND", "problem not found")
				return
			}
			items, total, err := subRepo.ListByProblem(ctx, id, page, perPage)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch submissions")
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"items":       items,
				"page":        page,
				"per_page":    perPage,
				"total_items": total,
				"total_pages": calcTotalPages(total, perPage),
			})
		})

		admin.POST("/users/bulk", func(c *gin.Context) {
			fileHeader, err := c.FormFile("file")
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "file フィールドに CSV を指定してください")
				return
			}
			file, err := fileHeader.Open()
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "ファイルを開けません")
				return
			}
			defer file.Close()

			reader := csv.NewReader(file)
			records, err := reader.ReadAll()
			if err != nil || len(records) == 0 {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "CSV を読み取れません")
				return
			}
			header := records[0]
			if len(header) < 2 || strings.ToLower(strings.TrimSpace(header[0])) != "userid" || strings.ToLower(strings.TrimSpace(header[1])) != "password" {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "ヘッダーは userid,password 形式にしてください")
				return
			}

			type failedRow struct {
				RowNumber int    `json:"row_number"`
				UserID    string `json:"userid"`
				Reason    string `json:"reason"`
			}
			var failed []failedRow
			created := 0

			ctx := c.Request.Context()
			for i, row := range records[1:] {
				rowNumber := i + 2 // header is row 1
				if len(row) < 2 {
					failed = append(failed, failedRow{RowNumber: rowNumber, UserID: "", Reason: "INVALID_ROW"})
					continue
				}
				userid := strings.TrimSpace(row[0])
				password := row[1]
				if userid == "" || password == "" {
					failed = append(failed, failedRow{RowNumber: rowNumber, UserID: userid, Reason: "VALIDATION_ERROR"})
					continue
				}
				hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if err != nil {
					failed = append(failed, failedRow{RowNumber: rowNumber, UserID: userid, Reason: "INTERNAL_ERROR"})
					continue
				}
				if _, err := userRepo.Create(ctx, userid, string(hash), "user"); err != nil {
					reason := "UNKNOWN_ERROR"
					if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
						reason = "USERID_ALREADY_EXISTS"
					}
					failed = append(failed, failedRow{RowNumber: rowNumber, UserID: userid, Reason: reason})
					continue
				}
				created++
			}

			c.JSON(http.StatusOK, gin.H{
				"created_count": created,
				"failed_count":  len(failed),
				"failed_rows":   failed,
			})
		})

		api.GET("/problems", func(c *gin.Context) {
			if _, ok := requireLogin(c); !ok {
				return
			}

			ctx := c.Request.Context()
			list, err := problemRepo.ListPublic(ctx)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch problems")
				return
			}
			c.JSON(http.StatusOK, gin.H{"problems": list})
		})

		api.GET("/problems/:id", func(c *gin.Context) {
			if _, ok := requireLogin(c); !ok {
				return
			}

			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
				return
			}
			ctx := c.Request.Context()
			detail, err := problemRepo.FindDetail(ctx, id)
			if err != nil {
				respondError(c, http.StatusNotFound, "NOT_FOUND", err.Error())
				return
			}
			statement := detail.StatementMD
			c.JSON(http.StatusOK, gin.H{
				"id":              detail.ID,
				"slug":            detail.Slug,
				"title":           detail.Title,
				"statement":       statement,
				"samples":         detail.Samples,
				"time_limit_ms":   detail.TimeLimitMS,
				"memory_limit_kb": detail.MemoryLimitKB,
			})
		})

		api.GET("/submissions", func(c *gin.Context) {
			sessionAny, _ := c.Get("session")
			sess, _ := sessionAny.(*sessions.Session)
			username, _ := sess.Values["userid"].(string)
			if strings.TrimSpace(username) == "" {
				respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "ログインが必要です。")
				return
			}

			page, perPage, err := parsePagination(c.Query("page"), c.Query("per_page"))
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
				return
			}

			var problemFilter *int64
			if pidStr := strings.TrimSpace(c.Query("problem_id")); pidStr != "" {
				pid, err := strconv.ParseInt(pidStr, 10, 64)
				if err != nil || pid <= 0 {
					respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "problem_id は正の整数で指定してください")
					return
				}
				problemFilter = &pid
			}

			ctx := c.Request.Context()
			user, err := userRepo.FindByUsername(ctx, username)
			if err != nil {
				respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "ユーザーが存在しません")
				return
			}

			items, total, err := subRepo.ListByUser(ctx, user.ID, problemFilter, page, perPage)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch submissions")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"items":       items,
				"page":        page,
				"per_page":    perPage,
				"total_items": total,
				"total_pages": calcTotalPages(total, perPage),
			})
		})

		api.GET("/problems/:id/submissions", func(c *gin.Context) {
			if _, ok := requireLogin(c); !ok {
				return
			}

			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil || id <= 0 {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
				return
			}

			page, perPage, err := parsePagination(c.Query("page"), c.Query("per_page"))
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
				return
			}

			ctx := c.Request.Context()
			isPublic, err := problemRepo.ExistsAndPublic(ctx, id)
			if err != nil {
				respondError(c, http.StatusNotFound, "NOT_FOUND", "problem not found")
				return
			}
			if !isPublic {
				respondError(c, http.StatusForbidden, "FORBIDDEN", "非公開の問題です")
				return
			}

			items, total, err := subRepo.ListByProblem(ctx, id, page, perPage)
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to fetch submissions")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"items":       items,
				"page":        page,
				"per_page":    perPage,
				"total_items": total,
				"total_pages": calcTotalPages(total, perPage),
			})
		})

		api.GET("/submissions/:id", func(c *gin.Context) {
			id, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id")
				return
			}
			ctx := c.Request.Context()
			res, err := subRepo.FindWithResult(ctx, id)
			if err != nil {
				respondError(c, http.StatusNotFound, "NOT_FOUND", "not found")
				return
			}

			// auth check: login required
			sessionAny, _ := c.Get("session")
			sess, _ := sessionAny.(*sessions.Session)
			userid, _ := sess.Values["userid"].(string)
			if strings.TrimSpace(userid) == "" {
				respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "ログインが必要です。")
				return
			}

			sourceCode := ""
			if strings.TrimSpace(res.SourcePath) != "" {
				if b, err := os.ReadFile(res.SourcePath); err == nil {
					sourceCode = string(b)
				} else {
					respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to read source code")
					return
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"id":            res.ID,
				"userid":        res.Username,
				"problem_id":    res.ProblemID,
				"problem_title": res.ProblemTitle,
				"language":      res.Language,
				"status":        res.Status,
				"verdict":       res.Verdict,
				"time_ms":       res.TimeMS,
				"memory_kb":     res.MemoryKB,
				"created_at":    res.CreatedAt,
				"updated_at":    res.UpdatedAt,
				"exit_code":     res.ExitCode,
				"error_message": res.ErrorMsg,
				"source_code":   sourceCode,
				"judge_details": res.Details,
			})
		})

		api.GET("/queue", func(c *gin.Context) {
			if _, ok := requireLogin(c); !ok {
				return
			}

			ctx := c.Request.Context()
			len, err := redisClient.LLen(ctx, PendingQueueKey).Result()
			if err != nil {
				respondError(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "failed to get queue length")
				return
			}
			c.JSON(http.StatusOK, gin.H{"pending": len})
		})
	}

	return r
}

func requireLogin(c *gin.Context) (string, bool) {
	sessionAny, _ := c.Get("session")
	sess, _ := sessionAny.(*sessions.Session)
	userid, _ := sess.Values["userid"].(string)
	if strings.TrimSpace(userid) == "" {
		respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "ログインが必要です。")
		return "", false
	}
	return userid, true
}

// ensureDir creates directory if not exists
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

var supportedLanguages = []map[string]string{
	{"key": "c", "label": "C (GCC)", "syntax": "c"},
	{"key": "cpp", "label": "C++17 (G++)", "syntax": "cpp"},
	{"key": "python", "label": "Python 3", "syntax": "python"},
	{"key": "java", "label": "Java 21", "syntax": "java"},
}

func isSupportedLanguage(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	for _, v := range supportedLanguages {
		if v["key"] == k {
			return true
		}
	}
	return false
}

// defaultSourceFor returns a short sample program per language for bulk test.
func defaultSourceFor(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "python":
		return "print('42')\n"
	case "java":
		return "public class Main{public static void main(String[]args){System.out.println(\"42\");}}\n"
	case "cpp":
		return "#include <bits/stdc++.h>\nusing namespace std;\nint main(){ios::sync_with_stdio(false);cin.tie(nullptr);long long a,b;if(!(cin>>a>>b))return 0;cout<<a+b<<\"\\n\";}\n"
	default: // c
		return "#include <stdio.h>\nint main(){printf(\"42\\n\");return 0;}\n"
	}
}

// createSubmissionWithSource inserts submission, writes source file, updates path, and enqueues.
func createSubmissionWithSource(ctx context.Context, cfg Config, subRepo SubmissionRepository, db *pgxpool.Pool, queue RedisClient, userID, problemID int64, lang, source string) (int64, error) {
	// Reserve ID
	subID, _, err := subRepo.Create(ctx, userID, problemID, lang, "")
	if err != nil {
		return 0, err
	}

	dir := filepath.Join(cfg.SubmissionDir, strconv.FormatInt(subID, 10))
	if err := ensureDir(dir); err != nil {
		_ = subRepo.Delete(ctx, subID)
		return 0, err
	}
	srcPath := filepath.Join(dir, "source")
	if err := os.WriteFile(srcPath, []byte(source), 0644); err != nil {
		_ = subRepo.Delete(ctx, subID)
		_ = os.RemoveAll(dir)
		return 0, err
	}
	if _, err := db.Exec(ctx, `UPDATE submissions SET source_path=$1 WHERE id=$2`, srcPath, subID); err != nil {
		_ = subRepo.Delete(ctx, subID)
		_ = os.RemoveAll(dir)
		return 0, err
	}
	if err := queue.Enqueue(ctx, PendingQueueKey, strconv.FormatInt(subID, 10)); err != nil {
		_ = subRepo.Delete(ctx, subID)
		_ = os.RemoveAll(dir)
		return 0, err
	}
	return subID, nil
}

const (
	defaultPerPage       = 20
	maxPerPage           = 100
	maxProblemImportSize = 8 * 1024 * 1024 // 8MB (upload payload limit)
)

func parsePagination(pageStr, perPageStr string) (int, int, error) {
	page := 1
	perPage := defaultPerPage
	if strings.TrimSpace(pageStr) != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p <= 0 {
			return 0, 0, errors.New("page は 1 以上の整数で指定してください")
		}
		page = p
	}
	if strings.TrimSpace(perPageStr) != "" {
		p, err := strconv.Atoi(perPageStr)
		if err != nil || p <= 0 {
			return 0, 0, errors.New("per_page は 1 以上の整数で指定してください")
		}
		if p > maxPerPage {
			p = maxPerPage
		}
		perPage = p
	}
	return page, perPage, nil
}

func calcTotalPages(total, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	return (total + perPage - 1) / perPage
}

func buildProblemTemplateZip() ([]byte, error) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	files := []struct {
		name    string
		content string
	}{
		{
			name: "two-string/problem.yaml",
			content: `slug: two-string
title: "Two String"

limits:
  time_ms: 2000
  memory_mb: 256

checker:
  type: exact
`,
		},
		{
			name:    "two-string/statement.md",
			content: "## 問題文\n2 行からなる入力で文字列 S, T が与えられます。S と T をこの順に連結した文字列を出力してください。\n\n## 制約\n- 1 ≤ |S| ≤ 100\n- 1 ≤ |T| ≤ 100\n- S, T は印字可能な ASCII 文字で構成される\n\n## 入力\n```\nS\nT\n```\n\n## 出力\n```\nS と T を連結した文字列を 1 行で出力せよ。\n```\n",
		},
		{name: "two-string/data/sample/01.in", content: "Hello\nOJ\n"},
		{name: "two-string/data/sample/01.out", content: "HelloOJ\n"},
		{name: "two-string/data/secret/01.in", content: "abc\nxyz\n"},
		{name: "two-string/data/secret/01.out", content: "abcxyz\n"},
	}

	for _, f := range files {
		w, err := zw.Create(f.name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(f.content)); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// buildProblemZipFromDB builds a problem archive from DB contents for admin download.
func buildProblemZipFromDB(detail ProblemDetail, cases []ProblemTestcase) ([]byte, error) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	write := func(name, content string) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(content))
		return err
	}

	problemYAML := fmt.Sprintf(`slug: %s
title: "%s"

limits:
  time_ms: %d
  memory_mb: %d

checker:
  type: %s
  eps: %g
`, detail.Slug, detail.Title, detail.TimeLimitMS, (detail.MemoryLimitKB+1023)/1024, defaultChecker(detail.CheckerType), detail.CheckerEps)

	if err := write(fmt.Sprintf("%s/problem.yaml", detail.Slug), problemYAML); err != nil {
		return nil, err
	}
	if err := write(fmt.Sprintf("%s/statement.md", detail.Slug), detail.StatementMD); err != nil {
		return nil, err
	}

	// write testcases
	sampleIdx, secretIdx := 1, 1
	for _, tc := range cases {
		prefix := "secret"
		idx := secretIdx
		if tc.IsSample {
			prefix = "sample"
			idx = sampleIdx
			sampleIdx++
		} else {
			secretIdx++
		}
		name := fmt.Sprintf("%02d", idx)
		if err := write(fmt.Sprintf("%s/data/%s/%s.in", detail.Slug, prefix, name), tc.InputText); err != nil {
			return nil, err
		}
		if err := write(fmt.Sprintf("%s/data/%s/%s.out", detail.Slug, prefix, name), tc.OutputText); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func defaultChecker(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "eps":
		return "eps"
	default:
		return "exact"
	}
}
