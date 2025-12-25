package core

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRecord represents a minimal projection stored in persistence layer.
type UserRecord struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

// AdminUserListItem is a projection for admin user listing (no password hash).
type AdminUserListItem struct {
	ID        int64     `json:"id"`
	Username  string    `json:"userid"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// UserRepository defines persistence operations for users.
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*UserRecord, error)
	Create(ctx context.Context, username, passwordHash, role string) (int64, error)
	HasAdmin(ctx context.Context) (bool, error)
	List(ctx context.Context, page, perPage int) ([]AdminUserListItem, int, error)
}

// PgUserRepository implements UserRepository using pgxpool.
type PgUserRepository struct {
	db *pgxpool.Pool
}

func NewPgUserRepository(db *pgxpool.Pool) *PgUserRepository {
	return &PgUserRepository{db: db}
}

func (r *PgUserRepository) FindByUsername(ctx context.Context, username string) (*UserRecord, error) {
	const q = `SELECT id, username, password_hash, role, created_at FROM users WHERE username=$1`
	var u UserRecord
	if err := r.db.QueryRow(ctx, q, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *PgUserRepository) Create(ctx context.Context, username, passwordHash, role string) (int64, error) {
	const q = `INSERT INTO users (username, password_hash, role) VALUES ($1,$2,$3) RETURNING id`
	var id int64
	if err := r.db.QueryRow(ctx, q, username, passwordHash, role).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *PgUserRepository) HasAdmin(ctx context.Context) (bool, error) {
	const q = `SELECT 1 FROM users WHERE role='admin' LIMIT 1`
	var one int
	if err := r.db.QueryRow(ctx, q).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// List returns paginated users without password hash.
func (r *PgUserRepository) List(ctx context.Context, page, perPage int) ([]AdminUserListItem, int, error) {
	if page <= 0 || perPage <= 0 {
		return nil, 0, errors.New("invalid pagination")
	}
	const countQ = `SELECT COUNT(*) FROM users`
	var total int
	if err := r.db.QueryRow(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id, username, role, created_at FROM users ORDER BY id LIMIT $1 OFFSET $2`, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]AdminUserListItem, 0, perPage)
	for rows.Next() {
		var u AdminUserListItem
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, u)
	}
	return items, total, rows.Err()
}
