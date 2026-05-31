package platform

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DatabaseURL        string
	RulesEnginePort    string
	TemplateEnginePort string
}

func LoadConfig() Config {
	return Config{
		DatabaseURL:        envOr("DATABASE_URL", "postgres://pav:pav@localhost:5432/pav?sslmode=disable"),
		RulesEnginePort:    envOr("RULES_ENGINE_PORT", "8081"),
		TemplateEnginePort: envOr("TEMPLATE_ENGINE_PORT", "8082"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}
