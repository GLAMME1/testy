package store

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

// создаёт пул соединений с Postgres с базовыми настройками
func NewPool(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
    cfg, err := pgxpool.ParseConfig(dbURL)
    if err != nil {
        return nil, err
    }
    cfg.MaxConns = 50
    cfg.MinConns = 10
    cfg.MaxConnLifetime = time.Hour
    cfg.MaxConnIdleTime = 30 * time.Minute
    return pgxpool.NewWithConfig(ctx, cfg)
}


