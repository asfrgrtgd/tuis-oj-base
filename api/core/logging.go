package core

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// SetupLogging configures log output to both stdout and a rotating file in cfg.LogDir.
// Caller should close the returned io.Closer on shutdown.
func SetupLogging(cfg Config, filename string) (io.Closer, error) {
	dir := cfg.LogDir
	if dir == "" {
		dir = "/var/log/oj"
	}
	if filename == "" {
		filename = "app.log"
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log dir %s: %w", dir, err)
	}

	path := filepath.Join(dir, filename)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", path, err)
	}

	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)
	gin.DefaultWriter = mw
	gin.DefaultErrorWriter = mw

	return f, nil
}
