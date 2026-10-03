package config

import "os"

// Config holds the settings the server needs to boot.
type Config struct {
	Port        string
	DatabaseURL string
}

// Load reads configuration from the environment, falling back to defaults
// suitable for local development.
func Load() Config {
	return Config{
		Port: env("PORT", "8080"),
		// Default points at the Postgres in deploy/docker-compose.yml.
		DatabaseURL: env("DATABASE_URL", "postgres://shortener:shortener@localhost:5435/shortener?sslmode=disable"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
