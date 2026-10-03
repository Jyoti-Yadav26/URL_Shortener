// Package service holds the shortening rules: validation, code generation,
// and expiry.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/Jyoti-Yadav26/URL_Shortener/internal/idgen"
	"github.com/Jyoti-Yadav26/URL_Shortener/internal/model"
	"github.com/Jyoti-Yadav26/URL_Shortener/internal/repository"
)

const (
	maxURLLength = 2048
	minAliasLen  = 3
	maxAliasLen  = 32
)

var aliasPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var (
	ErrInvalidURL    = errors.New("invalid url")
	ErrInvalidAlias  = errors.New("invalid alias")
	ErrInvalidExpiry = errors.New("invalid expiry")
	ErrAliasTaken    = errors.New("alias already taken")
	ErrNotFound      = errors.New("code not found")
	ErrExpired       = errors.New("link expired")
)

// URLRepository is the persistence this package needs. It is declared here,
// beside the code that calls it, so the service compiles against a behaviour
// rather than against pgx.
type URLRepository interface {
	NextID(ctx context.Context) (uint64, error)
	Insert(ctx context.Context, u model.URL) (model.URL, error)
	GetByCode(ctx context.Context, code string) (model.URL, error)
}

type Service struct {
	repo URLRepository
}

func New(repo URLRepository) *Service {
	return &Service{repo: repo}
}

// Shorten validates the request and stores a new short link. An empty alias
// means "generate a code"; expiresInHours of 0 means "never expires".
func (s *Service) Shorten(ctx context.Context, rawURL, alias string, expiresInHours int) (model.URL, error) {
	target, err := validateURL(rawURL)
	if err != nil {
		return model.URL{}, err
	}

	if alias != "" {
		if err := validateAlias(alias); err != nil {
			return model.URL{}, err
		}
	}

	expiresAt, err := expiryFromHours(expiresInHours)
	if err != nil {
		return model.URL{}, err
	}

	// Taken for the row id even when an alias supplies the code, so every
	// row costs exactly one sequence value.
	id, err := s.repo.NextID(ctx)
	if err != nil {
		return model.URL{}, fmt.Errorf("service: shorten: %w", err)
	}

	code := alias
	if code == "" {
		code = idgen.Encode(id)
	}

	stored, err := s.repo.Insert(ctx, model.URL{
		ID:        int64(id),
		Code:      code,
		TargetURL: target,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateCode) {
			return model.URL{}, fmt.Errorf("service: shorten %q: %w", code, ErrAliasTaken)
		}
		return model.URL{}, fmt.Errorf("service: shorten %q: %w", code, err)
	}

	return stored, nil
}

// Resolve finds the link a code points at, reporting ErrNotFound if there is
// no such code and ErrExpired if it has lapsed.
func (s *Service) Resolve(ctx context.Context, code string) (model.URL, error) {
	u, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.URL{}, fmt.Errorf("service: resolve %q: %w", code, ErrNotFound)
		}
		return model.URL{}, fmt.Errorf("service: resolve %q: %w", code, err)
	}

	if u.ExpiresAt != nil && !u.ExpiresAt.After(time.Now()) {
		return model.URL{}, fmt.Errorf("service: resolve %q: %w", code, ErrExpired)
	}

	return u, nil
}

func validateURL(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("service: url is empty: %w", ErrInvalidURL)
	}
	if len(raw) > maxURLLength {
		return "", fmt.Errorf("service: url is %d bytes, limit %d: %w", len(raw), maxURLLength, ErrInvalidURL)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("service: parse url: %w", ErrInvalidURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("service: scheme %q is not http or https: %w", u.Scheme, ErrInvalidURL)
	}
	if u.Hostname() == "" {
		return "", fmt.Errorf("service: url has no host: %w", ErrInvalidURL)
	}

	return u.String(), nil
}

func validateAlias(alias string) error {
	if len(alias) < minAliasLen || len(alias) > maxAliasLen {
		return fmt.Errorf("service: alias is %d characters, want %d to %d: %w",
			len(alias), minAliasLen, maxAliasLen, ErrInvalidAlias)
	}
	if !aliasPattern.MatchString(alias) {
		return fmt.Errorf("service: alias %q may only contain letters, digits, hyphen and underscore: %w",
			alias, ErrInvalidAlias)
	}
	return nil
}

func expiryFromHours(hours int) (*time.Time, error) {
	if hours == 0 {
		return nil, nil
	}
	if hours < 0 {
		return nil, fmt.Errorf("service: expires_in_hours is %d: %w", hours, ErrInvalidExpiry)
	}

	t := time.Now().Add(time.Duration(hours) * time.Hour)
	return &t, nil
}
