package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Jyoti-Yadav26/URL_Shortener/internal/model"
)

// TestURLRepository exercises Insert and GetByCode against a real Postgres.
// It is skipped unless TEST_DATABASE_URL is set, so `go test ./...` still
// passes on a machine with no database running.
func TestURLRepository(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; start deploy/docker-compose.yml and set it to run this test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(pool.Close)

	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "0001_create_urls.sql"))
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	repo := NewURLRepository(pool)

	// Unique per run so repeated runs never collide, and so the test does
	// not have to truncate a database it does not own.
	code := "it_" + time.Now().Format("20060102150405.000000000")
	target := "https://example.com/phase-three"

	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM urls WHERE code = $1`, code); err != nil {
			t.Errorf("cleanup %q: %v", code, err)
		}
	})

	t.Run("insert returns the stored row", func(t *testing.T) {
		id, err := repo.NextID(ctx)
		if err != nil {
			t.Fatalf("NextID: %v", err)
		}
		if id == 0 {
			t.Error("NextID returned 0, want a sequence value")
		}

		got, err := repo.Insert(ctx, model.URL{ID: int64(id), Code: code, TargetURL: target})
		if err != nil {
			t.Fatalf("Insert: %v", err)
		}
		if got.ID == 0 {
			t.Error("Insert returned ID 0, want the database-assigned id")
		}
		if got.Code != code {
			t.Errorf("Insert Code = %q, want %q", got.Code, code)
		}
		if got.TargetURL != target {
			t.Errorf("Insert TargetURL = %q, want %q", got.TargetURL, target)
		}
		if got.CreatedAt.IsZero() {
			t.Error("Insert returned zero CreatedAt, want the database default")
		}
	})

	t.Run("get by code round trips", func(t *testing.T) {
		got, err := repo.GetByCode(ctx, code)
		if err != nil {
			t.Fatalf("GetByCode: %v", err)
		}
		if got.Code != code || got.TargetURL != target {
			t.Errorf("GetByCode = %+v, want code %q and target %q", got, code, target)
		}
	})

	t.Run("duplicate code is rejected", func(t *testing.T) {
		id, err := repo.NextID(ctx)
		if err != nil {
			t.Fatalf("NextID: %v", err)
		}

		_, err = repo.Insert(ctx, model.URL{ID: int64(id), Code: code, TargetURL: target})
		if !errors.Is(err, ErrDuplicateCode) {
			t.Errorf("Insert duplicate error = %v, want ErrDuplicateCode", err)
		}
	})

	t.Run("missing code reports not found", func(t *testing.T) {
		_, err := repo.GetByCode(ctx, "definitely_not_stored")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("GetByCode missing error = %v, want ErrNotFound", err)
		}
	})
}
