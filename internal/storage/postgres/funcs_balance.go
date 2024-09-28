package postgres

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kTowkA/gophermart/internal/model"
	"github.com/kTowkA/gophermart/internal/storage"
)

func (p *PStorage) Balance(ctx context.Context, userID uuid.UUID) (model.ResponseBalance, error) {
	var (
		withdrawn float64
		accural   float64
	)
	err := p.QueryRow(
		ctx,
		`
		SELECT coalesce(SUM(withdrawals.sum),0) as withdrawn
		FROM withdrawals 
		WHERE user_id=$1
		`,
		userID,
	).Scan(&withdrawn)
	if err != nil {
		p.Error("получение списаний пользователя", slog.String("userID", userID.String()), slog.String("ошибка", err.Error()))
		return model.ResponseBalance{}, err
	}
	err = p.QueryRow(
		ctx,
		`
		SELECT coalesce(SUM(replenishments.sum),0) as replenishment
		FROM 
			replenishments,
			(
				SELECT order_id
				FROM orders 
				WHERE user_id=$1
			) AS orders
		WHERE replenishments.order_id=orders.order_id
		`,
		userID,
	).Scan(&accural)
	if err != nil {
		p.Error("получение пополенений пользователя", slog.String("userID", userID.String()), slog.String("ошибка", err.Error()))
		return model.ResponseBalance{}, err
	}
	p.Debug("успешное получение баланса у пользователя", slog.String("userID", userID.String()), slog.Float64("withdrawn", withdrawn), slog.Float64("current", accural-withdrawn))
	return model.ResponseBalance{
		Current:   accural - withdrawn,
		Withdrawn: withdrawn,
	}, nil
}

func (p *PStorage) Withdrawals(ctx context.Context, userID uuid.UUID) (model.ResponseWithdrawals, error) {
	rows, err := p.Query(
		ctx,
		`
		SELECT order_num,sum,withdrawn_at
		FROM withdrawals
		WHERE user_id=$1
		ORDER BY withdrawn_at DESC
		`,
		userID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		p.Warn("получение списаний пользователя. списаний нет", slog.String("userID", userID.String()))
		return model.ResponseWithdrawals{}, storage.ErrWithdrawalsNotFound
	}
	if err != nil {
		p.Warn("получение списаний пользователя.", slog.String("userID", userID.String()), slog.String("ошибка", err.Error()))
		return model.ResponseWithdrawals{}, err
	}
	defer rows.Close()
	withdrawals := make([]model.ResponseWithdraw, 0)
	for rows.Next() {
		withdrawal := model.ResponseWithdraw{}
		err = rows.Scan(
			&withdrawal.OrderNumber,
			&withdrawal.Sum,
			&withdrawal.ProcessedAt,
		)
		if err != nil {
			p.Warn("получение списания у пользователя", slog.String("userID", userID.String()), slog.String("ошибка", err.Error()))
			return model.ResponseWithdrawals{}, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	if len(withdrawals) == 0 {
		p.Warn("получение списаний пользователя. списаний нет", slog.String("userID", userID.String()))
		return model.ResponseWithdrawals{}, storage.ErrWithdrawalsNotFound
	}
	p.Debug("успешное получение списаний пользователя", slog.String("userID", userID.String()), slog.Int("всего списаний", len(withdrawals)))
	return withdrawals, nil
}

func (p *PStorage) Withdraw(ctx context.Context, userID uuid.UUID, requestWithdraw model.RequestWithdraw) error {

	balance, err := p.Balance(ctx, userID)
	if err != nil {
		return nil
	}
	if balance.Current < requestWithdraw.Sum {
		return storage.ErrWithdrawNotEnough
	}
	withdrawnID := uuid.New()
	_, err = p.Exec(
		ctx,
		`
		INSERT INTO withdrawals(withdrawn_id,order_num,sum,user_id,withdrawn_at) VALUES($1,$2,$3,$4,$5)
		`,
		withdrawnID,
		requestWithdraw.OrderNumber,
		requestWithdraw.Sum,
		userID,
		time.Now(),
	)
	if err != nil {
		p.Error("списание средств у пользователя", slog.String("userID", userID.String()), slog.String("ошибка", err.Error()))
	}
	p.Debug("успешное списание у пользователя", slog.String("userID", userID.String()), slog.String("списание в счет заказа", string(requestWithdraw.OrderNumber)), slog.Float64("сумма списания", requestWithdraw.Sum))
	return nil
}
