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

// Insert stores a new short link and returns the stored row, including the
// id and creation time the database assigned. It reports ErrDuplicateCode if
// code is already taken.
func (r *URLRepository) Insert(ctx context.Context, code, targetURL string) (model.URL, error) {
	const query = `
		INSERT INTO urls (code, target_url)
		VALUES ($1, $2)
		RETURNING id, code, target_url, created_at`

	var u model.URL
	err := r.pool.QueryRow(ctx, query, code, targetURL).
		Scan(&u.ID, &u.Code, &u.TargetURL, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return model.URL{}, fmt.Errorf("repository: insert %q: %w", code, ErrDuplicateCode)
		}
		return model.URL{}, fmt.Errorf("repository: insert %q: %w", code, err)
	}

	return u, nil
}

// GetByCode looks up a short link by its slug, reporting ErrNotFound if no
// row matches.
func (r *URLRepository) GetByCode(ctx context.Context, code string) (model.URL, error) {
	const query = `
		SELECT id, code, target_url, created_at
		FROM urls
		WHERE code = $1`

	var u model.URL
	err := r.pool.QueryRow(ctx, query, code).
		Scan(&u.ID, &u.Code, &u.TargetURL, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.URL{}, fmt.Errorf("repository: get %q: %w", code, ErrNotFound)
		}
		return model.URL{}, fmt.Errorf("repository: get %q: %w", code, err)
	}

	return u, nil
}
