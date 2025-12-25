package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Submission represents a user submission metadata stored in DB.
type Submission struct {
	ID         int64
	UserID     int64
	ProblemID  int64
	Language   string
	SourcePath string
	Status     string
	CreatedAt  time.Time
}

// SubmissionResult holds judge outcome.
type SubmissionResult struct {
	SubmissionID int64
	Verdict      string
	TimeMS       *int32
	MemoryKB     *int32
	StdoutPath   *string
	StderrPath   *string
	ExitCode     *int32
	ErrorMessage *string
	UpdatedAt    time.Time
	Details      []SubmissionJudgeDetail
}

// SubmissionJudgeDetail represents per-testcase execution detail.
type SubmissionJudgeDetail struct {
	Testcase string `json:"testcase"`
	Status   string `json:"status"`
	TimeMS   *int32 `json:"time_ms,omitempty"`
	MemoryKB *int32 `json:"memory_kb,omitempty"`
}

// SubmissionRepository defines persistence operations needed by worker/API.
type SubmissionRepository interface {
	FindByID(ctx context.Context, id int64) (*Submission, error)
	MarkStatus(ctx context.Context, id int64, status string) error
	SaveResult(ctx context.Context, result SubmissionResult, finalStatus string) error
	Create(ctx context.Context, userID, problemID int64, language, sourcePath string) (int64, time.Time, error)
	Delete(ctx context.Context, id int64) error
	FindWithResult(ctx context.Context, id int64) (*SubmissionResultView, error)
	AcquirePending(ctx context.Context, id int64) (*Submission, error)
	IncrementRetry(ctx context.Context, id int64) (int, error)
	CountByUser(ctx context.Context, userID int64) (int, error)
	CountSolvedProblemsByUser(ctx context.Context, userID int64) (int, error)
	ListByUser(ctx context.Context, userID int64, problemID *int64, page, perPage int) ([]SubmissionListItem, int, error)
	ListByProblem(ctx context.Context, problemID int64, page, perPage int) ([]SubmissionListItem, int, error)
}

// PgSubmissionRepository is a pgx implementation.
// NOTE: Expects tables `submissions` and `submission_results` to exist.
type PgSubmissionRepository struct {
	db *pgxpool.Pool
}

func NewPgSubmissionRepository(db *pgxpool.Pool) *PgSubmissionRepository {
	return &PgSubmissionRepository{db: db}
}

var ErrSubmissionNotPending = errors.New("submission not pending")

func (r *PgSubmissionRepository) FindByID(ctx context.Context, id int64) (*Submission, error) {
	const q = `SELECT id, user_id, problem_id, language, source_path, status, created_at FROM submissions WHERE id=$1`
	var s Submission
	if err := r.db.QueryRow(ctx, q, id).Scan(&s.ID, &s.UserID, &s.ProblemID, &s.Language, &s.SourcePath, &s.Status, &s.CreatedAt); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PgSubmissionRepository) MarkStatus(ctx context.Context, id int64, status string) error {
	if status == "" {
		return errors.New("status is empty")
	}
	const q = `UPDATE submissions SET status=$1, updated_at=NOW() WHERE id=$2`
	ct, err := r.db.Exec(ctx, q, status, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errors.New("submission not found")
	}
	return nil
}

func (r *PgSubmissionRepository) SaveResult(ctx context.Context, result SubmissionResult, finalStatus string) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const updStatus = `UPDATE submissions SET status=$1, updated_at=NOW() WHERE id=$2`
	if ct, err := tx.Exec(ctx, updStatus, finalStatus, result.SubmissionID); err != nil {
		return err
	} else if ct.RowsAffected() == 0 {
		return errors.New("submission not found")
	}

	const q = `INSERT INTO submission_results (submission_id, verdict, time_ms, memory_kb, stdout_path, stderr_path, exit_code, error_message, updated_at)
               VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
               ON CONFLICT (submission_id) DO UPDATE SET
                 verdict=EXCLUDED.verdict,
                 time_ms=EXCLUDED.time_ms,
                 memory_kb=EXCLUDED.memory_kb,
                 stdout_path=EXCLUDED.stdout_path,
                 stderr_path=EXCLUDED.stderr_path,
                 exit_code=EXCLUDED.exit_code,
                 error_message=EXCLUDED.error_message,
                 updated_at=NOW()`

	if _, err := tx.Exec(ctx, q, result.SubmissionID, result.Verdict, result.TimeMS, result.MemoryKB, result.StdoutPath, result.StderrPath, result.ExitCode, result.ErrorMessage); err != nil {
		return err
	}

	// refresh judge details
	if _, err := tx.Exec(ctx, `DELETE FROM submission_result_details WHERE submission_id=$1`, result.SubmissionID); err != nil {
		return err
	}
	for _, d := range result.Details {
		if _, err := tx.Exec(ctx, `INSERT INTO submission_result_details (submission_id, testcase, status, time_ms, memory_kb)
VALUES ($1,$2,$3,$4,$5)`, result.SubmissionID, d.Testcase, d.Status, d.TimeMS, d.MemoryKB); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *PgSubmissionRepository) Create(ctx context.Context, userID, problemID int64, language, sourcePath string) (int64, time.Time, error) {
	const q = `INSERT INTO submissions (user_id, problem_id, language, source_path, status)
			VALUES ($1,$2,$3,$4,'pending') RETURNING id, created_at`
	var id int64
	var created time.Time
	if err := r.db.QueryRow(ctx, q, userID, problemID, language, sourcePath).Scan(&id, &created); err != nil {
		return 0, time.Time{}, err
	}
	return id, created, nil
}

func (r *PgSubmissionRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM submissions WHERE id=$1`, id)
	return err
}

// AcquirePending locks a pending submission and transitions it to running atomically.
func (r *PgSubmissionRepository) AcquirePending(ctx context.Context, id int64) (*Submission, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	const sel = `SELECT id, user_id, problem_id, language, source_path, status, created_at FROM submissions WHERE id=$1 FOR UPDATE`
	var s Submission
	if err := tx.QueryRow(ctx, sel, id).Scan(&s.ID, &s.UserID, &s.ProblemID, &s.Language, &s.SourcePath, &s.Status, &s.CreatedAt); err != nil {
		return nil, err
	}
	if s.Status != "pending" {
		return nil, ErrSubmissionNotPending
	}

	const upd = `UPDATE submissions SET status='running', updated_at=NOW() WHERE id=$1`
	if _, err := tx.Exec(ctx, upd, id); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	s.Status = "running"
	return &s, nil
}

// IncrementRetry increments retry_count and returns the latest value.
func (r *PgSubmissionRepository) IncrementRetry(ctx context.Context, id int64) (int, error) {
	const q = `UPDATE submissions SET retry_count = retry_count + 1, updated_at=NOW() WHERE id=$1 RETURNING retry_count`
	var count int
	if err := r.db.QueryRow(ctx, q, id).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// CountByUser returns total submission count for user.
func (r *PgSubmissionRepository) CountByUser(ctx context.Context, userID int64) (int, error) {
	const q = `SELECT COUNT(*) FROM submissions WHERE user_id=$1`
	var c int
	if err := r.db.QueryRow(ctx, q, userID).Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

// CountSolvedProblemsByUser returns distinct accepted problem count (verdict AC).
func (r *PgSubmissionRepository) CountSolvedProblemsByUser(ctx context.Context, userID int64) (int, error) {
	const q = `SELECT COUNT(DISTINCT s.problem_id) FROM submissions s
LEFT JOIN submission_results r ON r.submission_id = s.id
WHERE s.user_id=$1 AND r.verdict='AC'`
	var c int
	if err := r.db.QueryRow(ctx, q, userID).Scan(&c); err != nil {
		return 0, err
	}
	return c, nil
}

// SubmissionResultView is a projection for API response.
type SubmissionResultView struct {
	ID           int64                   `json:"id"`
	UserID       int64                   `json:"user_id"`
	Username     string                  `json:"userid"`
	ProblemID    int64                   `json:"problem_id"`
	ProblemTitle string                  `json:"problem_title"`
	Language     string                  `json:"language"`
	Status       string                  `json:"status"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	Verdict      *string                 `json:"verdict"`
	TimeMS       *int32                  `json:"time_ms"`
	MemoryKB     *int32                  `json:"memory_kb"`
	StdoutPath   *string                 `json:"stdout_path"`
	StderrPath   *string                 `json:"stderr_path"`
	ExitCode     *int32                  `json:"exit_code"`
	ErrorMsg     *string                 `json:"error_message"`
	SourcePath   string                  `json:"-"`
	Details      []SubmissionJudgeDetail `json:"judge_details"`
}

// SubmissionListItem is a flattened view for list endpoints.
type SubmissionListItem struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Username     string    `json:"userid"`
	ProblemID    int64     `json:"problem_id"`
	ProblemTitle string    `json:"problem_title,omitempty"`
	Language     string    `json:"language"`
	Status       string    `json:"status"`
	Verdict      *string   `json:"verdict"`
	TimeMS       *int32    `json:"time_ms"`
	MemoryKB     *int32    `json:"memory_kb"`
	CreatedAt    time.Time `json:"created_at"`
}

func (r *PgSubmissionRepository) FindWithResult(ctx context.Context, id int64) (*SubmissionResultView, error) {
	const q = `
SELECT s.id, s.user_id, u.username, s.problem_id, p.title, s.language, s.status, s.source_path,
       s.created_at, s.updated_at,
       sr.verdict, sr.time_ms, sr.memory_kb, sr.stdout_path, sr.stderr_path, sr.exit_code, sr.error_message
FROM submissions s
JOIN users u ON u.id = s.user_id
JOIN problems p ON p.id = s.problem_id
LEFT JOIN submission_results sr ON sr.submission_id = s.id
WHERE s.id=$1`
	var v SubmissionResultView
	var verdict, stdoutPath, stderrPath, errMsg sql.NullString
	var timeMS, memoryKB sql.NullInt32
	var exitCode sql.NullInt32
	if err := r.db.QueryRow(ctx, q, id).Scan(
		&v.ID, &v.UserID, &v.Username, &v.ProblemID, &v.ProblemTitle, &v.Language, &v.Status, &v.SourcePath,
		&v.CreatedAt, &v.UpdatedAt,
		&verdict, &timeMS, &memoryKB, &stdoutPath, &stderrPath, &exitCode, &errMsg,
	); err != nil {
		return nil, err
	}
	if verdict.Valid {
		v.Verdict = &verdict.String
	}
	if timeMS.Valid {
		v.TimeMS = ptrInt32(timeMS.Int32)
	}
	if memoryKB.Valid {
		v.MemoryKB = ptrInt32(memoryKB.Int32)
	}
	if stdoutPath.Valid {
		v.StdoutPath = &stdoutPath.String
	}
	if stderrPath.Valid {
		v.StderrPath = &stderrPath.String
	}
	if exitCode.Valid {
		v.ExitCode = ptrInt32(exitCode.Int32)
	}
	if errMsg.Valid {
		v.ErrorMsg = &errMsg.String
	}

	// load judge details (if any)
	const detailQ = `SELECT testcase, status, time_ms, memory_kb FROM submission_result_details WHERE submission_id=$1 ORDER BY id`
	rows, err := r.db.Query(ctx, detailQ, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tc, status string
		var t sql.NullInt32
		var m sql.NullInt32
		if err := rows.Scan(&tc, &status, &t, &m); err != nil {
			return nil, err
		}
		v.Details = append(v.Details, SubmissionJudgeDetail{
			Testcase: tc,
			Status:   status,
			TimeMS:   ptrFromNullInt32(t),
			MemoryKB: ptrFromNullInt32(m),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PgSubmissionRepository) ListByUser(ctx context.Context, userID int64, problemID *int64, page, perPage int) ([]SubmissionListItem, int, error) {
	if page <= 0 || perPage <= 0 {
		return nil, 0, errors.New("invalid pagination")
	}

	filters := []string{"s.user_id=$1"}
	args := []interface{}{userID}
	if problemID != nil && *problemID > 0 {
		filters = append(filters, fmt.Sprintf("s.problem_id=$%d", len(args)+1))
		args = append(args, *problemID)
	}
	where := strings.Join(filters, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM submissions s WHERE %s`, where)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitPlaceholder := len(args) + 1
	offsetPlaceholder := len(args) + 2
	query := fmt.Sprintf(`
SELECT s.id, s.user_id, u.username, s.problem_id, p.title, s.language, s.status,
       sr.verdict, sr.time_ms, sr.memory_kb, s.created_at
FROM submissions s
JOIN users u ON u.id = s.user_id
JOIN problems p ON p.id = s.problem_id
LEFT JOIN submission_results sr ON sr.submission_id = s.id
WHERE %s
ORDER BY s.created_at DESC
LIMIT $%d OFFSET $%d`, where, limitPlaceholder, offsetPlaceholder)

	argsWithPage := append(append([]interface{}{}, args...), perPage, (page-1)*perPage)
	rows, err := r.db.Query(ctx, query, argsWithPage...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]SubmissionListItem, 0, perPage)
	for rows.Next() {
		var v SubmissionListItem
		if err := rows.Scan(&v.ID, &v.UserID, &v.Username, &v.ProblemID, &v.ProblemTitle, &v.Language, &v.Status, &v.Verdict, &v.TimeMS, &v.MemoryKB, &v.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, v)
	}
	return items, total, rows.Err()
}

func (r *PgSubmissionRepository) ListByProblem(ctx context.Context, problemID int64, page, perPage int) ([]SubmissionListItem, int, error) {
	if page <= 0 || perPage <= 0 {
		return nil, 0, errors.New("invalid pagination")
	}

	const countQuery = `SELECT COUNT(*) FROM submissions WHERE problem_id=$1`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, problemID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
SELECT s.id, s.user_id, u.username, s.problem_id, p.title, s.language, s.status,
       sr.verdict, sr.time_ms, sr.memory_kb, s.created_at
FROM submissions s
JOIN users u ON u.id = s.user_id
JOIN problems p ON p.id = s.problem_id
LEFT JOIN submission_results sr ON sr.submission_id = s.id
WHERE s.problem_id=$1
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, problemID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]SubmissionListItem, 0, perPage)
	for rows.Next() {
		var v SubmissionListItem
		if err := rows.Scan(&v.ID, &v.UserID, &v.Username, &v.ProblemID, &v.ProblemTitle, &v.Language, &v.Status, &v.Verdict, &v.TimeMS, &v.MemoryKB, &v.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, v)
	}
	return items, total, rows.Err()
}

func ptrInt32(v int32) *int32 {
	return &v
}

func ptrFromNullInt32(n sql.NullInt32) *int32 {
	if !n.Valid {
		return nil
	}
	return ptrInt32(n.Int32)
}
