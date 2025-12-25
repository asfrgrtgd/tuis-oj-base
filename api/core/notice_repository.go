package core

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Notice struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NoticeRepository interface {
	List(ctx context.Context, page, perPage int) ([]Notice, int, error)
	Get(ctx context.Context, id int64) (*Notice, error)
	Create(ctx context.Context, title, body string) (*Notice, error)
	Update(ctx context.Context, id int64, title, body string) (*Notice, error)
	Delete(ctx context.Context, id int64) error
}

type PgNoticeRepository struct {
	db *pgxpool.Pool
}

func NewPgNoticeRepository(db *pgxpool.Pool) *PgNoticeRepository {
	return &PgNoticeRepository{db: db}
}

func (r *PgNoticeRepository) List(ctx context.Context, page, perPage int) ([]Notice, int, error) {
	if page <= 0 || perPage <= 0 {
		return nil, 0, errors.New("invalid pagination")
	}
	const countQ = `SELECT COUNT(*) FROM notices`
	var total int
	if err := r.db.QueryRow(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `
SELECT id, title, body, created_at, updated_at
FROM notices
ORDER BY updated_at DESC, id DESC
LIMIT $1 OFFSET $2
`, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Notice, 0, perPage)
	for rows.Next() {
		var n Notice
		if err := rows.Scan(&n.ID, &n.Title, &n.Body, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, n)
	}
	return items, total, rows.Err()
}

func (r *PgNoticeRepository) Get(ctx context.Context, id int64) (*Notice, error) {
	const q = `SELECT id, title, body, created_at, updated_at FROM notices WHERE id=$1`
	var n Notice
	if err := r.db.QueryRow(ctx, q, id).Scan(&n.ID, &n.Title, &n.Body, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *PgNoticeRepository) Create(ctx context.Context, title, body string) (*Notice, error) {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	const q = `INSERT INTO notices (title, body) VALUES ($1,$2) RETURNING id, created_at, updated_at`
	var n Notice
	if err := r.db.QueryRow(ctx, q, title, body).Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return nil, err
	}
	n.Title = title
	n.Body = body
	return &n, nil
}

func (r *PgNoticeRepository) Update(ctx context.Context, id int64, title, body string) (*Notice, error) {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	const q = `UPDATE notices SET title=$1, body=$2 WHERE id=$3 RETURNING id, created_at, updated_at`
	var n Notice
	if err := r.db.QueryRow(ctx, q, title, body, id).Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt); err != nil {
		return nil, err
	}
	n.Title = title
	n.Body = body
	return &n, nil
}

func (r *PgNoticeRepository) Delete(ctx context.Context, id int64) error {
	const q = `DELETE FROM notices WHERE id=$1`
	_, err := r.db.Exec(ctx, q, id)
	return err
}
