package core

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// BootstrapAdmin creates an initial admin user when none exists.
// It is idempotent: if an admin already exists, it does nothing.
func BootstrapAdmin(ctx context.Context, repo UserRepository, cfg Config) error {
	if !cfg.BootstrapAdminEnabled {
		return nil
	}

	has, err := repo.HasAdmin(ctx)
	if err != nil {
		return err
	}
	if has {
		return nil
	}

	username := "admin"
	password, err := generatePassword(32)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if _, err := repo.Create(ctx, username, string(hash), "admin"); err != nil {
		return err
	}

	if cfg.InitialAdminPasswordPath != "" {
		if err := os.WriteFile(cfg.InitialAdminPasswordPath, []byte(password+"\n"), 0o600); err != nil {
			return err
		}
		log.Printf("initial admin created; credentials written to %s", cfg.InitialAdminPasswordPath)
	} else {
		log.Printf("initial admin created username=%s password=%s", username, password)
	}

	return nil
}

func generatePassword(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("password length must be positive")
	}
	// base64 encoding: need 3/4 overhead; ensure enough bytes
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw)[:length], nil
}
