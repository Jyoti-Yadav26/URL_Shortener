// Package model holds the domain types shared across the layers.
package model

import "time"

// URL is a shortened link: Code is the slug that appears in the short link,
// TargetURL is where it redirects to.
type URL struct {
	ID        int64
	Code      string
	TargetURL string
	CreatedAt time.Time
}
