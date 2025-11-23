// Package postgresql предоставляет реализации репозиториев поверх PostgreSQL
package postgresql

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool создает новый пул подключений к PostgreSQL
func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}
	return pool, nil
}
