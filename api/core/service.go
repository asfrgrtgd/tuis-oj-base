package core

import (
	"context"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// RepositoryAuthService is a placeholder that will wrap repository and hashing.
type RepositoryAuthService struct {
	users UserRepository
}

// NewRepositoryAuthService is a stub constructor for future real implementation.
func NewRepositoryAuthService(users UserRepository) *RepositoryAuthService {
	return &RepositoryAuthService{users: users}
}

// Authenticate is a placeholder that delegates to repository until hashing is implemented.
func (s *RepositoryAuthService) Authenticate(username, password string) (User, error) {
	if strings.TrimSpace(username) == "" || password == "" {
		return User{}, ErrInvalidCredentials
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	u, err := s.users.FindByUsername(ctx, username)
	if err != nil || u == nil {
		return User{}, ErrInvalidCredentials
	}

	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return User{}, ErrInvalidCredentials
	}
	return User{
		ID:        u.ID,
		Username:  u.Username,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}, nil
}
