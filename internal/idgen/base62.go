// Package idgen converts unsigned integers to and from base62, turning a
// URL's numeric database id into a short slug.
package idgen

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

const (
	alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	base     = uint64(len(alphabet))

	// maxEncodedLen is how many base62 digits math.MaxUint64 needs.
	maxEncodedLen = 11
)

var (
	ErrEmptyInput       = errors.New("empty input")
	ErrInvalidCharacter = errors.New("invalid base62 character")
	ErrOverflow         = errors.New("value overflows uint64")
)

// Encode renders n in base62. The result never has leading zeros, except for
// n == 0, which encodes as the single digit "0".
func Encode(n uint64) string {
	if n == 0 {
		return alphabet[:1]
	}

	var buf [maxEncodedLen]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = alphabet[n%base]
		n /= base
	}
	return string(buf[i:])
}

// Decode parses a base62 string. It reports ErrEmptyInput, ErrInvalidCharacter,
// or ErrOverflow for input it cannot represent as a uint64.
func Decode(s string) (uint64, error) {
	if s == "" {
		return 0, fmt.Errorf("idgen: decode: %w", ErrEmptyInput)
	}

	var n uint64
	for i := 0; i < len(s); i++ {
		d := strings.IndexByte(alphabet, s[i])
		if d < 0 {
			return 0, fmt.Errorf("idgen: decode %q: %w", s, ErrInvalidCharacter)
		}
		if n > (math.MaxUint64-uint64(d))/base {
			return 0, fmt.Errorf("idgen: decode %q: %w", s, ErrOverflow)
		}
		n = n*base + uint64(d)
	}
	return n, nil
}
