package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Jyoti-Yadav26/URL_Shortener/internal/idgen"
	"github.com/Jyoti-Yadav26/URL_Shortener/internal/model"
	"github.com/Jyoti-Yadav26/URL_Shortener/internal/repository"
)

// fakeRepo is a hand-written in-memory stand-in for URLRepository. It behaves
// like the real store for the parts the service cares about: ids come from a
// counter, codes are unique, and a missing code is ErrNotFound.
type fakeRepo struct {
	nextID uint64
	rows   map[string]model.URL
}

func newFakeRepo(seed ...model.URL) *fakeRepo {
	f := &fakeRepo{nextID: 1, rows: make(map[string]model.URL)}
	for _, u := range seed {
		f.rows[u.Code] = u
	}
	return f
}

func (f *fakeRepo) NextID(ctx context.Context) (uint64, error) {
	id := f.nextID
	f.nextID++
	return id, nil
}

func (f *fakeRepo) Insert(ctx context.Context, u model.URL) (model.URL, error) {
	if _, taken := f.rows[u.Code]; taken {
		return model.URL{}, fmt.Errorf("fake: %w", repository.ErrDuplicateCode)
	}
	u.CreatedAt = time.Now()
	f.rows[u.Code] = u
	return u, nil
}

func (f *fakeRepo) GetByCode(ctx context.Context, code string) (model.URL, error) {
	u, ok := f.rows[code]
	if !ok {
		return model.URL{}, fmt.Errorf("fake: %w", repository.ErrNotFound)
	}
	return u, nil
}

func TestShorten(t *testing.T) {
	tests := []struct {
		name     string
		seed     []model.URL
		url      string
		alias    string
		hours    int
		wantCode string
		wantErr  error
	}{
		{
			name:     "new link gets a generated code",
			url:      "https://example.com/a/long/path",
			wantCode: idgen.Encode(1),
		},
		{
			name:     "alias is used as the code",
			url:      "https://example.com",
			alias:    "my-link",
			wantCode: "my-link",
		},
		{
			name:    "taken alias is rejected",
			seed:    []model.URL{{Code: "taken", TargetURL: "https://example.com"}},
			url:     "https://example.com",
			alias:   "taken",
			wantErr: ErrAliasTaken,
		},
		{
			name:    "non http scheme is rejected",
			url:     "ftp://example.com/file",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "url without a host is rejected",
			url:     "https://",
			wantErr: ErrInvalidURL,
		},
		{
			name:    "alias with a dot is rejected",
			url:     "https://example.com",
			alias:   "no.dots",
			wantErr: ErrInvalidAlias,
		},
		{
			name:    "alias that is too short is rejected",
			url:     "https://example.com",
			alias:   "ab",
			wantErr: ErrInvalidAlias,
		},
		{
			name:    "negative expiry is rejected",
			url:     "https://example.com",
			hours:   -1,
			wantErr: ErrInvalidExpiry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := New(newFakeRepo(tt.seed...))

			got, err := svc.Shorten(context.Background(), tt.url, tt.alias, tt.hours)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Shorten error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			if got.Code != tt.wantCode {
				t.Errorf("Shorten Code = %q, want %q", got.Code, tt.wantCode)
			}
			if got.TargetURL != tt.url {
				t.Errorf("Shorten TargetURL = %q, want %q", got.TargetURL, tt.url)
			}
			if got.ExpiresAt != nil {
				t.Errorf("Shorten ExpiresAt = %v, want nil for hours = 0", got.ExpiresAt)
			}
		})
	}
}

func TestShortenSetsExpiry(t *testing.T) {
	svc := New(newFakeRepo())

	before := time.Now()
	got, err := svc.Shorten(context.Background(), "https://example.com", "", 24)
	if err != nil {
		t.Fatalf("Shorten: %v", err)
	}

	if got.ExpiresAt == nil {
		t.Fatal("Shorten ExpiresAt = nil, want a time about 24h out")
	}
	gap := got.ExpiresAt.Sub(before)
	if gap < 23*time.Hour+59*time.Minute || gap > 24*time.Hour+time.Minute {
		t.Errorf("Shorten ExpiresAt is %v from now, want about 24h", gap)
	}
}

func TestResolve(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name       string
		seed       []model.URL
		code       string
		wantTarget string
		wantErr    error
	}{
		{
			name:       "live link resolves",
			seed:       []model.URL{{Code: "abc", TargetURL: "https://example.com/live"}},
			code:       "abc",
			wantTarget: "https://example.com/live",
		},
		{
			name:       "link with a future expiry resolves",
			seed:       []model.URL{{Code: "abc", TargetURL: "https://example.com/live", ExpiresAt: &future}},
			code:       "abc",
			wantTarget: "https://example.com/live",
		},
		{
			name:    "unknown code is not found",
			code:    "nope",
			wantErr: ErrNotFound,
		},
		{
			name:    "past expiry is gone",
			seed:    []model.URL{{Code: "old", TargetURL: "https://example.com/old", ExpiresAt: &past}},
			code:    "old",
			wantErr: ErrExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := New(newFakeRepo(tt.seed...))

			got, err := svc.Resolve(context.Background(), tt.code)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Resolve error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}

			if got.TargetURL != tt.wantTarget {
				t.Errorf("Resolve TargetURL = %q, want %q", got.TargetURL, tt.wantTarget)
			}
		})
	}
}
