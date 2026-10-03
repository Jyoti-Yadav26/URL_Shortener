package config

import "os"

// Config holds the settings the server needs to boot.
type Config struct {
	Port string
}

// Load reads configuration from the environment, falling back to defaults
// suitable for local development.
func Load() Config {
	return Config{
		Port: env("PORT", "8080"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
