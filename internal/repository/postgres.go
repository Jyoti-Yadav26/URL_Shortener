// Package repository persists URLs in Postgres.
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool opens a connection pool and verifies it can reach the database.
// The caller owns the pool and must Close it.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("repository: new pool: %w", err)
	}

	// pgxpool.New is lazy, so without this a bad DSN or an unreachable
	// server would only surface on the first query.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("repository: ping: %w", err)
	}

	return pool, nil
}
