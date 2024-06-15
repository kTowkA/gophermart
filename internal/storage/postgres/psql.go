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
	"github.com/kTowkA/gophermart/internal/model"
	"github.com/kTowkA/gophermart/internal/storage"
)

//go:embed migrations/*.sql
var fs embed.FS

type PStorage struct {
	*pgxpool.Pool
	*slog.Logger
}

func New(ctx context.Context, pdns string, logger *logger.Log) (*PStorage, error) {
	sl := logger.WithGroup("postgres")
	err := migrations(pdns)
	if err != nil {
		sl.Error("проведение миграций", slog.String("ошибка", err.Error()))
		return nil, err
	}
	pool, err := pgxpool.New(ctx, pdns)
	if err != nil {
		sl.Error("создание нового пула соединений", slog.String("err", err.Error()))
		return nil, err
	}
	ps := PStorage{
		Pool:   pool,
		Logger: sl,
	}
	statuses := []*model.Status{
		&storage.StatusUndefined,
		&storage.StatusNew,
		&storage.StatusRegistered,
		&storage.StatusInvalid,
		&storage.StatusProcessing,
		&storage.StatusProcessed}
	err = ps.SaveStatuses(ctx, statuses)
	if err != nil {
		ps.Close()
		sl.Error("сохранение статусов", slog.String("ошибка", err.Error()))
		return nil, err
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

func (p *PStorage) SaveStatuses(ctx context.Context, statuses []*model.Status) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		p.Error("создание транзакции", slog.String("ошибка", err.Error()))
		return err
	}
	for i := range statuses {
		var statusID int
		err = tx.QueryRow(
			ctx,
			"INSERT INTO statuses(value) VALUES($1) RETURNING status_id",
			statuses[i].Value(),
		).Scan(&statusID)
		if err != nil {
			p.Error("создание статуса", slog.String("статус", statuses[i].Value()), slog.String("ошибка", err.Error()))
			return tx.Rollback(ctx)
		}
		statuses[i].SetKey(statusID)
	}
	err = tx.Commit(ctx)
	if err != nil {
		p.Error("сохранение изменений. создание статусов", slog.String("ошибка", err.Error()))
		return err
	}
	return nil
}
func (p *PStorage) Close() error {
	p.Pool.Close()
	return nil
}
