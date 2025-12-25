package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gorilla/sessions"

	"tuis-oj-prototype/core"
)

func main() {
	cfg := core.Load()
	ctx := context.Background()

	logCloser, err := core.SetupLogging(cfg, "api.log")
	if err != nil {
		log.Fatalf("failed to setup logging: %v", err)
	}
	defer logCloser.Close()

	db, err := core.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	redisClient, err := core.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}
	defer redisClient.Close()

	// Ensure writable dir for submissions
	if cfg.SubmissionDir == "" {
		log.Fatalf("submission dir path is empty")
	}
	if abs, err := filepath.Abs(cfg.SubmissionDir); err == nil {
		cfg.SubmissionDir = abs
	}
	if err := os.MkdirAll(cfg.SubmissionDir, 0o755); err != nil {
		log.Fatalf("failed to ensure submission dir %s: %v", cfg.SubmissionDir, err)
	}

	// Gorilla cookie store for session management.
	store := sessions.NewCookieStore([]byte(cfg.SessionKey))

	userRepo := core.NewPgUserRepository(db)
	authService := core.NewRepositoryAuthService(userRepo)

	if err := core.BootstrapAdmin(ctx, userRepo, cfg); err != nil {
		log.Fatalf("bootstrap admin failed: %v", err)
	}

	router := core.NewRouter(cfg, store, authService, db, redisClient)

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("starting api server on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
