package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Jyoti-Yadav26/URL_Shortener/internal/model"
)

var (
	ErrNotFound      = errors.New("url not found")
	ErrDuplicateCode = errors.New("code already in use")
)

// uniqueViolation is the SQLSTATE Postgres returns when a UNIQUE constraint
// is violated.
const uniqueViolation = "23505"

type URLRepository struct {
	pool *pgxpool.Pool
}

func NewURLRepository(pool *pgxpool.Pool) *URLRepository {
	return &URLRepository{pool: pool}
}

// NextID draws the next value from the urls id sequence. Callers use it to
// derive a code before inserting, then pass the same id to Insert so the row
// and its code stay in step.
func (r *URLRepository) NextID(ctx context.Context) (uint64, error) {
	const query = `SELECT nextval(pg_get_serial_sequence('urls', 'id'))`

	var id uint64
	if err := r.pool.QueryRow(ctx, query).Scan(&id); err != nil {
		return 0, fmt.Errorf("repository: next id: %w", err)
	}
	return id, nil
}

// Insert stores a new short link and returns the stored row, including the
// creation time the database assigned. It reports ErrDuplicateCode if the
// code is already taken.
func (r *URLRepository) Insert(ctx context.Context, u model.URL) (model.URL, error) {
	const query = `
		INSERT INTO urls (id, code, target_url, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, code, target_url, created_at, expires_at`

	var out model.URL
	err := r.pool.QueryRow(ctx, query, u.ID, u.Code, u.TargetURL, u.ExpiresAt).
		Scan(&out.ID, &out.Code, &out.TargetURL, &out.CreatedAt, &out.ExpiresAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return model.URL{}, fmt.Errorf("repository: insert %q: %w", u.Code, ErrDuplicateCode)
		}
		return model.URL{}, fmt.Errorf("repository: insert %q: %w", u.Code, err)
	}

	return out, nil
}

// GetByCode looks up a short link by its slug, reporting ErrNotFound if no
// row matches.
func (r *URLRepository) GetByCode(ctx context.Context, code string) (model.URL, error) {
	const query = `
		SELECT id, code, target_url, created_at, expires_at
		FROM urls
		WHERE code = $1`

	var u model.URL
	err := r.pool.QueryRow(ctx, query, code).
		Scan(&u.ID, &u.Code, &u.TargetURL, &u.CreatedAt, &u.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.URL{}, fmt.Errorf("repository: get %q: %w", code, ErrNotFound)
		}
		return model.URL{}, fmt.Errorf("repository: get %q: %w", code, err)
	}

	return u, nil
}
