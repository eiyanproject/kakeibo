package config

import "os"

type Config struct {
	DatabaseURL   string
	Addr          string
	SessionSecure bool
}

func Load() Config {
	return Config{
		DatabaseURL:   getenv("DATABASE_URL", "postgres://kakeibo:kakeibo@localhost:5432/kakeibo?sslmode=disable"),
		Addr:          getenv("ADDR", ":8080"),
		SessionSecure: getenv("SESSION_SECURE", "false") == "true",
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
