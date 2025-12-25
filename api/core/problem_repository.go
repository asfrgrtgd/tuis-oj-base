package core

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProblemRepository interface {
	ExistsAndPublic(ctx context.Context, id int64) (bool, error)
	Exists(ctx context.Context, id int64) (bool, error)
	ListPublic(ctx context.Context) ([]ProblemMeta, error)
	FindDetail(ctx context.Context, id int64) (*ProblemDetail, error)
	FindDetailAdmin(ctx context.Context, id int64) (*ProblemDetail, error)
	ListTestcases(ctx context.Context, id int64) ([]ProblemTestcase, error)
	CreateWithTestcases(ctx context.Context, input ProblemCreateInput) (int64, error)
	UpdateProblem(ctx context.Context, id int64, input ProblemUpdateInput) error
	AdminList(ctx context.Context, page, perPage int) ([]ProblemAdminListItem, int, error)
	ProblemStats(ctx context.Context, id int64) (*ProblemStats, error)
}

type PgProblemRepository struct {
	db *pgxpool.Pool
}

func NewPgProblemRepository(db *pgxpool.Pool) *PgProblemRepository {
	return &PgProblemRepository{db: db}
}

func (r *PgProblemRepository) ExistsAndPublic(ctx context.Context, id int64) (bool, error) {
	const q = `SELECT is_public FROM problems WHERE id=$1`
	var isPublic bool
	if err := r.db.QueryRow(ctx, q, id).Scan(&isPublic); err != nil {
		return false, err
	}
	return isPublic, nil
}

func (r *PgProblemRepository) Exists(ctx context.Context, id int64) (bool, error) {
	const q = `SELECT 1 FROM problems WHERE id=$1`
	var one int
	if err := r.db.QueryRow(ctx, q, id).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type ProblemMeta struct {
	ID            int64  `json:"id"`
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	TimeLimitMS   int32  `json:"time_limit_ms"`
	MemoryLimitKB int32  `json:"memory_limit_kb"`
}

type ProblemDetail struct {
	ProblemMeta
	StatementMD string // inline markdown
	Samples     []SampleCase
	CheckerType string
	CheckerEps  float64
}

type SampleCase struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

// ProblemAdminListItem represents admin-visible problem summary with counts.
type ProblemAdminListItem struct {
	ID              int64  `json:"id"`
	Slug            string `json:"slug"`
	Title           string `json:"title"`
	Visibility      string `json:"visibility"`
	SolvedCount     int    `json:"solved_count"`
	SubmissionCount int    `json:"submission_count"`
}

// ProblemStats aggregates submission statistics for a problem.
type ProblemStats struct {
	ProblemID           int64          `json:"problem_id"`
	Title               string         `json:"title"`
	SubmissionCount     int            `json:"submission_count"`
	AcceptedCount       int            `json:"accepted_count"`
	UniqueUsers         int            `json:"unique_users"`
	UniqueAcceptedUsers int            `json:"unique_accepted_users"`
	AcceptanceRate      float64        `json:"acceptance_rate"`
	LastSubmissionAt    *time.Time     `json:"last_submission_at"`
	StatusBreakdown     map[string]int `json:"status_breakdown"`
}

// ProblemTestcase represents a single testcase path pair.
type ProblemTestcase struct {
	InputPath  string
	OutputPath string
	InputText  string
	OutputText string
	IsSample   bool
}

// ProblemCreateInput represents a new problem and all testcases to be inserted atomically.
type ProblemCreateInput struct {
	Title         string
	Slug          string
	StatementMD   string
	StatementPath *string
	TimeLimitMS   int32
	MemoryLimitKB int32
	IsPublic      bool
	CheckerType   string
	CheckerEps    float64
	Testcases     []ProblemTestcaseInput
}

// ProblemTestcaseInput holds inline testcase content for creation.
type ProblemTestcaseInput struct {
	InputText  string
	OutputText string
	InputPath  string
	OutputPath string
	IsSample   bool
}

// ProblemUpdateInput holds mutable fields for a problem.
type ProblemUpdateInput struct {
	Title         *string
	StatementMD   *string
	TimeLimitMS   *int32
	MemoryLimitKB *int32
	IsPublic      *bool
	CheckerType   *string
	CheckerEps    *float64
}

func (r *PgProblemRepository) ListPublic(ctx context.Context) ([]ProblemMeta, error) {
	const q = `SELECT id, slug, title, time_limit_ms, memory_limit_kb FROM problems WHERE is_public = TRUE ORDER BY id`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProblemMeta
	for rows.Next() {
		var p ProblemMeta
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.TimeLimitMS, &p.MemoryLimitKB); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// AdminList returns all problems (公開/非公開含む) with submission counts.
func (r *PgProblemRepository) AdminList(ctx context.Context, page, perPage int) ([]ProblemAdminListItem, int, error) {
	if page <= 0 || perPage <= 0 {
		return nil, 0, errors.New("invalid pagination")
	}

	const countQ = `SELECT COUNT(*) FROM problems`
	var total int
	if err := r.db.QueryRow(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, err
	}

	const q = `
SELECT p.id, p.slug, p.title, p.is_public,
       COALESCE(SUM(CASE WHEN sr.verdict='AC' THEN 1 ELSE 0 END),0) AS solved_count,
       COALESCE(COUNT(s.id),0) AS submission_count
FROM problems p
LEFT JOIN submissions s ON s.problem_id = p.id
LEFT JOIN submission_results sr ON sr.submission_id = s.id
GROUP BY p.id
ORDER BY p.id
LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, q, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []ProblemAdminListItem
	for rows.Next() {
		var item ProblemAdminListItem
		var isPublic bool
		if err := rows.Scan(&item.ID, &item.Slug, &item.Title, &isPublic, &item.SolvedCount, &item.SubmissionCount); err != nil {
			return nil, 0, err
		}
		if isPublic {
			item.Visibility = "public"
		} else {
			item.Visibility = "hidden"
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (r *PgProblemRepository) findDetail(ctx context.Context, id int64, allowHidden bool) (*ProblemDetail, bool, error) {
	const q = `SELECT id, slug, title, statement_md, time_limit_ms, memory_limit_kb, is_public, checker_type, checker_eps FROM problems WHERE id=$1`
	var d ProblemDetail
	var isPublic bool
	var statementMD *string
	var checkerType string
	var checkerEps float64
	if err := r.db.QueryRow(ctx, q, id).Scan(&d.ID, &d.Slug, &d.Title, &statementMD, &d.TimeLimitMS, &d.MemoryLimitKB, &isPublic, &checkerType, &checkerEps); err != nil {
		log.Printf("findDetail problem query err id=%d: %v", id, err)
		return nil, false, err
	}
	if !allowHidden && !isPublic {
		return nil, isPublic, errors.New("problem not public")
	}
	d.CheckerType = strings.TrimSpace(checkerType)
	d.CheckerEps = checkerEps

	// sample testcases (fallback if older schema lacks inline columns)
	const t = `SELECT input_path, output_path, input_text, output_text FROM testcases WHERE problem_id=$1 AND is_sample=TRUE ORDER BY id`
	rows, err := r.db.Query(ctx, t, id)
	if err != nil && (strings.Contains(err.Error(), "input_text") || strings.Contains(err.Error(), "output_text")) {
		rows, err = r.db.Query(ctx, `SELECT input_path, output_path, NULL::TEXT AS input_text, NULL::TEXT AS output_text FROM testcases WHERE problem_id=$1 AND is_sample=TRUE ORDER BY id`, id)
	}
	if err != nil {
		log.Printf("findDetail sample query err id=%d: %v", id, err)
		return nil, isPublic, err
	}
	defer rows.Close()
	for rows.Next() {
		var inPath, outPath, inText, outText sql.NullString
		if err := rows.Scan(&inPath, &outPath, &inText, &outText); err != nil {
			log.Printf("findDetail sample scan err id=%d: %v", id, err)
			return nil, isPublic, err
		}
		inStr := strings.TrimSpace(inText.String)
		outStr := strings.TrimSpace(outText.String)
		if outStr == "" {
			return nil, isPublic, errors.New("sample testcase output missing; inline text required")
		}
		d.Samples = append(d.Samples, SampleCase{Input: inStr, Output: outStr})
	}
	if statementMD != nil {
		d.StatementMD = *statementMD
	}
	return &d, isPublic, rows.Err()
}

func (r *PgProblemRepository) FindDetail(ctx context.Context, id int64) (*ProblemDetail, error) {
	d, _, err := r.findDetail(ctx, id, false)
	return d, err
}

// FindDetailAdmin returns problem detail regardless of visibility.
func (r *PgProblemRepository) FindDetailAdmin(ctx context.Context, id int64) (*ProblemDetail, error) {
	d, _, err := r.findDetail(ctx, id, true)
	return d, err
}

// ListTestcases returns all testcases (including hidden) for the problem in deterministic order.
func (r *PgProblemRepository) ListTestcases(ctx context.Context, id int64) ([]ProblemTestcase, error) {
	const q = `SELECT input_path, output_path, input_text, output_text, is_sample FROM testcases WHERE problem_id=$1 ORDER BY id`
	rows, err := r.db.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ProblemTestcase
	for rows.Next() {
		var inPath, outPath, inText, outText sql.NullString
		var isSample bool
		if err := rows.Scan(&inPath, &outPath, &inText, &outText, &isSample); err != nil {
			return nil, err
		}
		tc := ProblemTestcase{
			InputPath:  inPath.String,
			OutputPath: outPath.String,
			InputText:  inText.String,
			OutputText: outText.String,
			IsSample:   isSample,
		}
		if strings.TrimSpace(tc.OutputText) == "" {
			return nil, errors.New("testcase output missing; file path fallback disabled")
		}
		out = append(out, tc)
	}
	return out, rows.Err()
}

// ProblemStats aggregates submission statistics for a problem.
func (r *PgProblemRepository) ProblemStats(ctx context.Context, id int64) (*ProblemStats, error) {
	const summaryQ = `
SELECT p.title,
       COALESCE(COUNT(s.id),0) AS submission_count,
       COALESCE(SUM(CASE WHEN sr.verdict='AC' THEN 1 ELSE 0 END),0) AS accepted_count,
       COALESCE(COUNT(DISTINCT s.user_id),0) AS unique_users,
       COALESCE(COUNT(DISTINCT CASE WHEN sr.verdict='AC' THEN s.user_id END),0) AS unique_accepted_users,
       MAX(s.created_at) AS last_submission_at
FROM problems p
LEFT JOIN submissions s ON s.problem_id = p.id
LEFT JOIN submission_results sr ON sr.submission_id = s.id
WHERE p.id=$1
GROUP BY p.id`
	var stats ProblemStats
	var lastSub sql.NullTime
	if err := r.db.QueryRow(ctx, summaryQ, id).Scan(
		&stats.Title, &stats.SubmissionCount, &stats.AcceptedCount, &stats.UniqueUsers, &stats.UniqueAcceptedUsers, &lastSub,
	); err != nil {
		return nil, err
	}
	stats.ProblemID = id
	if lastSub.Valid {
		stats.LastSubmissionAt = &lastSub.Time
	}
	if stats.SubmissionCount > 0 {
		stats.AcceptanceRate = float64(stats.AcceptedCount) / float64(stats.SubmissionCount)
	}

	// breakdown
	const breakdownQ = `SELECT COALESCE(sr.verdict,'UNKNOWN') AS verdict, COUNT(*) FROM submissions s LEFT JOIN submission_results sr ON sr.submission_id = s.id WHERE s.problem_id=$1 GROUP BY verdict`
	rows, err := r.db.Query(ctx, breakdownQ, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats.StatusBreakdown = map[string]int{}
	for rows.Next() {
		var verdict string
		var count int
		if err := rows.Scan(&verdict, &count); err != nil {
			return nil, err
		}
		stats.StatusBreakdown[verdict] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &stats, nil
}

// CreateWithTestcases inserts a problem and all its testcases in a single transaction.
func (r *PgProblemRepository) CreateWithTestcases(ctx context.Context, input ProblemCreateInput) (int64, error) {
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Slug) == "" {
		return 0, errors.New("title and slug are required")
	}
	if len(input.Testcases) == 0 {
		return 0, errors.New("at least one testcase is required")
	}
	if strings.TrimSpace(input.CheckerType) == "" {
		input.CheckerType = "exact"
	}
	input.CheckerType = strings.ToLower(strings.TrimSpace(input.CheckerType))
	if input.CheckerType != "exact" && input.CheckerType != "eps" {
		return 0, errors.New("checker_type must be exact or eps")
	}
	if input.CheckerType == "eps" && input.CheckerEps <= 0 {
		return 0, errors.New("checker_eps must be > 0 when checker_type=eps")
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var problemID int64
	if err := tx.QueryRow(ctx, `INSERT INTO problems (slug, title, statement_path, statement_md, time_limit_ms, memory_limit_kb, is_public, checker_type, checker_eps)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		input.Slug, input.Title, input.StatementPath, input.StatementMD, input.TimeLimitMS, input.MemoryLimitKB, input.IsPublic, input.CheckerType, input.CheckerEps).Scan(&problemID); err != nil {
		return 0, err
	}

	for _, tc := range input.Testcases {
		if strings.TrimSpace(tc.InputText) == "" || strings.TrimSpace(tc.OutputText) == "" {
			return 0, errors.New("testcase input/output is required")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO testcases (problem_id, input_path, output_path, input_text, output_text, is_sample)
VALUES ($1,$2,$3,$4,$5,$6)`, problemID, nonNilString(tc.InputPath), nonNilString(tc.OutputPath), tc.InputText, tc.OutputText, tc.IsSample); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return problemID, nil
}

func nonNilString(v string) string {
	if v == "" {
		return ""
	}
	return v
}

// UpdateProblem updates mutable fields of a problem.
func (r *PgProblemRepository) UpdateProblem(ctx context.Context, id int64, input ProblemUpdateInput) error {
	var sets []string
	var args []any

	if input.Title != nil {
		sets = append(sets, "title=$"+strconv.Itoa(len(args)+1))
		args = append(args, strings.TrimSpace(*input.Title))
	}
	if input.StatementMD != nil {
		sets = append(sets, "statement_md=$"+strconv.Itoa(len(args)+1))
		args = append(args, *input.StatementMD)
	}
	if input.TimeLimitMS != nil {
		if *input.TimeLimitMS <= 0 {
			return errors.New("time_limit_ms must be > 0")
		}
		sets = append(sets, "time_limit_ms=$"+strconv.Itoa(len(args)+1))
		args = append(args, *input.TimeLimitMS)
	}
	if input.MemoryLimitKB != nil {
		if *input.MemoryLimitKB <= 0 {
			return errors.New("memory_limit_kb must be > 0")
		}
		sets = append(sets, "memory_limit_kb=$"+strconv.Itoa(len(args)+1))
		args = append(args, *input.MemoryLimitKB)
	}
	if input.IsPublic != nil {
		sets = append(sets, "is_public=$"+strconv.Itoa(len(args)+1))
		args = append(args, *input.IsPublic)
	}
	if input.CheckerType != nil {
		ct := strings.ToLower(strings.TrimSpace(*input.CheckerType))
		if ct != "exact" && ct != "eps" {
			return errors.New("checker_type must be exact or eps")
		}
		sets = append(sets, "checker_type=$"+strconv.Itoa(len(args)+1))
		args = append(args, ct)
	}
	if input.CheckerEps != nil {
		if input.CheckerType != nil && strings.ToLower(strings.TrimSpace(*input.CheckerType)) == "eps" && *input.CheckerEps <= 0 {
			return errors.New("checker_eps must be > 0 when checker_type=eps")
		}
		sets = append(sets, "checker_eps=$"+strconv.Itoa(len(args)+1))
		args = append(args, *input.CheckerEps)
	}

	if len(sets) == 0 {
		return nil
	}
	args = append(args, id)
	q := "UPDATE problems SET " + strings.Join(sets, ", ") + " WHERE id=$" + strconv.Itoa(len(args))
	_, err := r.db.Exec(ctx, q, args...)
	return err
}
