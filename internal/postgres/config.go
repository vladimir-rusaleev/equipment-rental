package postgres

import (
	"fmt"
	"net"
	"net/url"
	"os"

	"rent/internal/platform"
)

func Storage() (string, error) {
	storage := platform.Env("STORAGE", "memory")
	if storage != "memory" && storage != "postgres" {
		return "", fmt.Errorf("unsupported STORAGE %q: use memory or postgres", storage)
	}
	return storage, nil
}

// DSN permits a complete per-service URL or builds one from local/Compose settings.
func DSN(variable, database string) (string, error) {
	if value := os.Getenv(variable); value != "" {
		return value, nil
	}
	password := os.Getenv("APP_DB_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("set %s or APP_DB_PASSWORD for PostgreSQL", variable)
	}
	address := net.JoinHostPort(platform.Env("POSTGRES_HOST", "127.0.0.1"), platform.Env("POSTGRES_PORT", "5432"))
	dsn := url.URL{Scheme: "postgres", User: url.UserPassword(database, password), Host: address, Path: database}
	query := dsn.Query()
	query.Set("sslmode", platform.Env("POSTGRES_SSLMODE", "disable"))
	query.Set("timezone", "UTC")
	dsn.RawQuery = query.Encode()
	return dsn.String(), nil
}
