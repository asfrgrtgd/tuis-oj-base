package core

import (
	"errors"
	"time"
)

// User represents an authenticated principal returned to handlers.
type User struct {
	ID        int64
	Username  string
	Role      string
	CreatedAt time.Time
}

var (
	// ErrInvalidCredentials is returned when userid/password is wrong.
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// AuthService defines authentication behaviour.
type AuthService interface {
	Authenticate(username, password string) (User, error)
}
