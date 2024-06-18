package postgres

import (
	"context"
	"errors"
	"log/slog"

	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kTowkA/gophermart/internal/logger"
	"github.com/kTowkA/gophermart/internal/model"
	"github.com/kTowkA/gophermart/internal/storage"
)

type PStorage struct {
	*pgxpool.Pool
	*slog.Logger
}

// NewStorage создает новое хранилище типа PStorage, реализующее интерфейс storage.Storage
func NewStorage(ctx context.Context, connString string, logger *logger.Log) (*PStorage, error) {
	sl := logger.WithGroup("postgres")
	pool, err := pgxpool.New(ctx, connString)
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
		&storage.StatusProcessed,
	}
	err = ps.SaveStatuses(ctx, statuses)
	if err != nil {
		ps.Close(ctx)
		sl.Error("сохранение/обновление статусов", slog.String("ошибка", err.Error()))
		return nil, err
	}
	return &ps, nil
}

func (p *PStorage) SaveStatuses(ctx context.Context, statuses []*model.Status) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		p.Error("создание транзакции", slog.String("ошибка", err.Error()))
		return err
	}
	for i := range statuses {
		var statusID int
		err = tx.QueryRow(ctx, "SELECT status_id FROM statuses WHERE value=$1", statuses[i].Value()).Scan(&statusID)
		if err == nil {
			statuses[i].SetKey(statusID)
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			p.Error("получение статуса", slog.String("статус", statuses[i].Value()), slog.String("ошибка", err.Error()))
			return tx.Rollback(ctx)
		}
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
func (p *PStorage) Close(ctx context.Context) error {
	p.Pool.Close()
	return nil
}
