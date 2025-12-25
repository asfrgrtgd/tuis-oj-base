package core

import (
	"os"
	"strconv"
	"strings"
)

// Config holds runtime settings for the API process.
type Config struct {
	Port                     string   // HTTP listen port (e.g., "3000")
	SessionKey               string   // Cookie signing/encryption key
	CookieSecure             bool     // Whether to set Secure flag on session cookie
	CookieSameSite           string   // SameSite policy: Strict/Lax/None
	LogDir                   string   // Directory to write application logs
	DatabaseURL              string   // PostgreSQL DSN
	RedisURL                 string   // Redis URL (redis://host:port/db)
	GoJudgeURL               string   // go-judge HTTP endpoint base
	CSRFSecret               string   // secret for CSRF token generation/validation
	SubmissionDir            string   // base directory to store submission files
	WorkerConcurrency        int      // number of worker goroutines (= go-judge parallelism)
	InitialAdminPasswordPath string   // where to write generated admin password (if empty -> log output)
	BootstrapAdminEnabled    bool     // whether to run bootstrap admin creation at startup
	AllowedOrigins           []string // allowed origins for CORS/CSRF origin check
	CompileTimeLimitMs       int      // per-language compile time limit passed to go-judge
}

// Load populates Config from environment variables with sane defaults.
func Load() Config {
	return Config{
		Port:           firstNonEmpty(os.Getenv("PORT"), "3000"),
		SessionKey:     firstNonEmpty(os.Getenv("SESSION_KEY"), "change-this-session-key"),
		CookieSecure:   boolFromEnv("COOKIE_SECURE", false),
		CookieSameSite: firstNonEmpty(os.Getenv("COOKIE_SAMESITE"), "Strict"),
		LogDir:         firstNonEmpty(os.Getenv("LOG_DIR"), "/var/log/oj"),
		DatabaseURL:    firstNonEmpty(os.Getenv("DATABASE_URL"), os.Getenv("POSTGRES_URL"), "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"),
		RedisURL:       firstNonEmpty(os.Getenv("REDIS_URL"), "redis://localhost:6379/0"),
		GoJudgeURL:     firstNonEmpty(os.Getenv("GOJUDGE_URL"), "http://localhost:5050"),
		CSRFSecret:     firstNonEmpty(os.Getenv("CSRF_SECRET"), "change-this-csrf-secret"),
		SubmissionDir:  firstNonEmpty(os.Getenv("SUBMISSION_DIR"), "./submission-files"),
		WorkerConcurrency: intFromEnv("WORKER_CONCURRENCY",
			intFromEnv("GOJUDGE_PARALLELISM", 4)),
		InitialAdminPasswordPath: firstNonEmpty(os.Getenv("INITIAL_ADMIN_PASSWORD_PATH"), "/run/oj-secrets/initial_admin_password.secret"),
		BootstrapAdminEnabled:    boolFromEnv("BOOTSTRAP_ADMIN", true),
		AllowedOrigins:           parseCSV(os.Getenv("ALLOWED_ORIGINS")),
		CompileTimeLimitMs:       intFromEnv("COMPILE_TIME_LIMIT_MS", 5000),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// boolFromEnv reads a boolean from env var name, falling back to defaultVal when empty or invalid.
func boolFromEnv(name string, defaultVal bool) bool {
	if v := os.Getenv(name); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}

// intFromEnv reads an int from env var name, falling back to defaultVal when empty or invalid.
func intFromEnv(name string, defaultVal int) int {
	if v := os.Getenv(name); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

// parseCSV splits comma-separated list and trims spaces; empty entries are skipped.
func parseCSV(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if t := strings.TrimSpace(v); t != "" {
			out = append(out, t)
		}
	}
	return out
}
