package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kTowkA/gophermart/internal/logger"
)

//go:embed migrations/*.sql
var fs embed.FS

type PStorage struct {
	*pgxpool.Pool
	*slog.Logger
}

func New(ctx context.Context, pdns string, logger *logger.Log) (*PStorage, error) {
	sl := logger.WithGroup("postgres")
	pool, err := pgxpool.New(ctx, pdns)
	if err != nil {
		sl.Error("создание нового пула соединений", slog.String("err", err.Error()))
		return nil, err
	}
	err = migrations(pdns)
	if err != nil {
		sl.Error("проведение миграций", slog.String("ошибка", err.Error()))
		return nil, err
	}
	ps := PStorage{
		Pool:   pool,
		Logger: sl,
	}
	return &ps, nil
}

func migrations(pdns string) error {
	d, err := iofs.New(fs, "migrations")
	if err != nil {
		return fmt.Errorf("создание драйвера для считывания миграций. %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", d, strings.Replace(pdns, "postgres", "pgx5", 1))
	if err != nil {
		return fmt.Errorf("создание экземпляра миграций. %w", err)
	}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("применение миграций. %w", err)
	}
	return nil
}

func (p *PStorage) Close() error {
	p.Pool.Close()
	return nil
}
