package idgen

import (
	"errors"
	"math"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name string
		in   uint64
		want string
	}{
		{name: "zero", in: 0, want: "0"},
		{name: "last digit", in: 9, want: "9"},
		{name: "first uppercase", in: 10, want: "A"},
		{name: "last uppercase", in: 35, want: "Z"},
		{name: "first lowercase", in: 36, want: "a"},
		{name: "last lowercase", in: 61, want: "z"},
		{name: "rolls over to two digits", in: 62, want: "10"},
		{name: "largest two digit value", in: 3843, want: "zz"},
		{name: "rolls over to three digits", in: 3844, want: "100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Encode(tt.in); got != tt.want {
				t.Errorf("Encode(%d) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    uint64
		wantErr error
	}{
		{name: "zero", in: "0", want: 0},
		{name: "last lowercase", in: "z", want: 61},
		{name: "two digits", in: "10", want: 62},
		{name: "leading zeros are ignored", in: "001", want: 1},
		{name: "empty", in: "", wantErr: ErrEmptyInput},
		{name: "punctuation", in: "ab-c", wantErr: ErrInvalidCharacter},
		{name: "non ascii", in: "aé", wantErr: ErrInvalidCharacter},
		{name: "eleven digits overflow", in: "zzzzzzzzzzz", wantErr: ErrOverflow},
		{name: "twelve digits overflow", in: "100000000000", wantErr: ErrOverflow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Decode(%q) error = %v, want %v", tt.in, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Decode(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   uint64
	}{
		{name: "zero", in: 0},
		{name: "one", in: 1},
		{name: "base boundary", in: 62},
		{name: "typical id", in: 1234567890},
		{name: "max uint64", in: math.MaxUint64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := Encode(tt.in)

			got, err := Decode(encoded)
			if err != nil {
				t.Fatalf("Decode(%q) returned error: %v", encoded, err)
			}
			if got != tt.in {
				t.Errorf("round trip of %d went through %q and came back as %d", tt.in, encoded, got)
			}
		})
	}
}
