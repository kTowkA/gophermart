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

func (p *PStorage) SaveOrder(ctx context.Context, userID uuid.UUID, orderNum model.OrderNumber) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		p.Error("создание транзакции", slog.String("ошибка", err.Error()))
		return err
	}
	var userIDintoDB uuid.UUID
	err = tx.QueryRow(
		ctx,
		"SELECT user_id FROM orders WHERE order_num=$1",
		string(orderNum),
	).Scan(&userIDintoDB)
	if err == nil {
		if userIDintoDB == userID {
			p.Warn("поиск заказа по переданному orderNum. пользователь уже загружал заказ", slog.String("номер заказа", string(orderNum)))
			_ = tx.Rollback(ctx)
			return storage.ErrOrderWasAlreadyUpload
		}
		p.Warn("поиск заказа по переданному orderNum. заказ загружал другой пользователь", slog.String("номер заказа", string(orderNum)))
		_ = tx.Rollback(ctx)
		return storage.ErrOrderWasUploadByAnotherUser
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		p.Error("поиск пользователя создающего заказ", slog.String("номер заказа", string(orderNum)), slog.String("ошибка", err.Error()))
		return err
	}
	orderID := uuid.New()
	_, err = tx.Exec(
		ctx,
		"INSERT INTO orders(order_id,order_num,user_id,status_id,adding_at,update_at) VALUES($1,$2,$3,$4,$5,$6)",
		orderID,
		string(orderNum),
		userID,
		storage.StatusNew.Key(),
		time.Now(),
		time.Now(),
	)
	if err != nil {
		p.Error("сохранение заказа", slog.String("номер заказа", string(orderNum)), slog.String("пользователь", userID.String()), slog.String("ошибка", err.Error()))
		_ = tx.Rollback(ctx)
		return err
	}

	_, err = tx.Exec(
		ctx,
		"INSERT INTO orders_statuses(order_id,status_id,adding_at,update_at) VALUES($1,$2,$3,$4)",
		orderID,
		storage.StatusNew.Key(),
		time.Now(),
		time.Now(),
	)
	if err != nil {
		p.Error("сохранение связи статус-заказ", slog.String("номер заказа", string(orderNum)), slog.String("статус", storage.StatusNew.Value()), slog.String("пользователь", userID.String()), slog.String("ошибка", err.Error()))
		_ = tx.Rollback(ctx)
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		p.Error("сохранение заказа. фиксация изменений", slog.String("номер заказа", string(orderNum)), slog.String("пользователь", userID.String()), slog.String("ошибка", err.Error()))
		return err
	}
	return nil
}

func (p *PStorage) UpdateOrders(ctx context.Context, info []model.ResponseAccuralSystem) (int, error) {
	tx, err := p.Begin(ctx)
	if err != nil {
		return 0, err
	}
	b := pgx.Batch{}
	for _, new := range info {
		// если был завершен расчет то сохраняем в таблице пополнений
		if new.Status.Value() == storage.StatusProcessed.Value() {
			replenishmentID := uuid.New()
			b.Queue(
				`INSERT INTO replenishments(replenishment_id,order_id,sum,replenishment_at) 
				VALUES(
					$1,
					(SELECT order_id FROM orders WHERE order_num=$2),
					$3,
					$4)
				`,
				replenishmentID,
				string(new.OrderNumber),
				new.Accrual,
				time.Now(),
			)
		}
		// здесь обновляем таблицу заказов
		b.Queue(
			"UPDATE orders SET status_id=$1 WHERE order_num=$2",
			storage.StatusByValue(new.Status.Value()).Key(),
			string(new.OrderNumber),
		)
		// и в конце обновляем связи статусы-заказы
		b.Queue(
			"UPDATE orders_statuses SET status_id=$1,adding_at=$2,update_at=$3 WHERE order_id=(SELECT order_id FROM orders WHERE order_num=$4)",
			storage.StatusByValue(new.Status.Value()).Key(),
			time.Now(),
			time.Now(),
			string(new.OrderNumber),
		)
	}
	br := tx.SendBatch(ctx, &b)
	err = br.Close()
	if err != nil {
		p.Error("закрытие пакета с данными", slog.String("ошибка", err.Error()))
		_ = tx.Rollback(ctx)
		return 0, err
	}
	err = tx.Commit(ctx)
	if err != nil {
		p.Error("сохранение изменений", slog.String("ошибка", err.Error()))
		_ = tx.Rollback(ctx)
		return 0, err
	}
	return len(info), nil
}

func (p *PStorage) UpdateOrder(ctx context.Context, info model.ResponseAccuralSystem) error {
	var (
		orderID  uuid.UUID
		statusID int
	)
	err := p.QueryRow(ctx, "SELECT order_id,status_id FROM orders WHERE order_num=$1", info.OrderNumber).Scan(&orderID, &statusID)
	if errors.Is(err, pgx.ErrNoRows) {
		p.Warn("поиск ID заказа по переданному номеру. заказа с таким номером нет", slog.String("номер заказа", string(info.OrderNumber)))
		return storage.ErrOrdersNotFound
	}
	if err != nil {
		p.Error("поиск ID заказа по переданному номеру", slog.String("номер заказа", string(info.OrderNumber)), slog.String("ошибка", err.Error()))
		return err
	}
	if statusID == storage.StatusByValue(info.Status.Value()).Key() {
		p.Warn("обновление заказа. данные актуальны", slog.String("номер заказа", string(info.OrderNumber)))
		return storage.ErrNothingHasBeenDone
	}
	_, err = p.UpdateOrders(ctx, []model.ResponseAccuralSystem{info})
	return err
}

func (p *PStorage) OrdersByStatuses(ctx context.Context, statuses []model.Status, limit, offset int) (model.ResponseOrders, error) {
	statusesValues := make([]string, len(statuses))
	for i := range statuses {
		statusesValues[i] = statuses[i].Value()
	}
	rows, err := p.Query(
		ctx,
		`
		SELECT order_num
		FROM orders
		WHERE status_id  = ANY (SELECT status_id FROM statuses WHERE value = ANY ($1))
		LIMIT $2
		OFFSET $3
		`,
		statusesValues,
		limit,
		offset,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		p.Warn("поиск заказов по статусам. заказов нет.", slog.Int("лимит", limit), slog.Int("смещение", offset))
		return nil, storage.ErrOrdersNotFound
	}
	if err != nil {
		p.Error("поиск заказов по статусам.", slog.String("ошибка", err.Error()))
		return nil, err
	}
	defer rows.Close()
	orders := make([]model.ResponseOrder, 0)
	for rows.Next() {
		order := model.ResponseOrder{}
		err = rows.Scan(
			&order.OrderNumber,
		)
		if err != nil {
			p.Error("получение номера заказа", slog.String("ошибка", err.Error()))
			return nil, err
		}
		orders = append(orders, order)
	}
	if len(orders) == 0 {
		p.Warn("поиск заказов по статусам. заказов нет.", slog.Int("лимит", limit), slog.Int("смещение", offset))
		return nil, storage.ErrOrdersNotFound
	}
	return orders, nil
}

func (p *PStorage) Orders(ctx context.Context, userID uuid.UUID) (model.ResponseOrders, error) {
	rows, err := p.Query(
		ctx,
		`
		SELECT 
			orders.order_num,orders.status,coalesce(replenishments.sum,0),orders.adding_at
		FROM 
    		(
				SELECT orders.order_id,orders.order_num,orders.user_id,orders.adding_at,orders.update_at,statuses.value as status
    			FROM orders,statuses
    			WHERE orders.status_id=statuses.status_id AND orders.user_id=$1
			) as orders
		LEFT JOIN replenishments ON orders.order_id=replenishments.order_id
		ORDER BY orders.adding_at DESC
		`,
		userID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		p.Warn("поиск заказов у пользователя. заказов нет.", slog.String("userID", userID.String()))
		return nil, storage.ErrOrdersNotFound
	}
	if err != nil {
		p.Error("поиск заказов у пользователя.", slog.String("userID", userID.String()), slog.String("ошибка", err.Error()))
		return nil, err
	}
	defer rows.Close()
	orders := make([]model.ResponseOrder, 0)
	for rows.Next() {
		order := model.ResponseOrder{}
		statusVal := ""
		err = rows.Scan(
			&order.OrderNumber,
			&statusVal,
			&order.Accrual,
			&order.UploadedAt,
		)
		if err != nil {
			p.Error("поиск заказов у пользователя.", slog.String("userID", userID.String()), slog.String("ошибка", err.Error()))
			return nil, err
		}
		order.Status = storage.StatusByValue(statusVal)
		p.Debug(
			"найден заказа",
			slog.String("userID", userID.String()),
			slog.String("номер", string(order.OrderNumber)),
			slog.String("статус", order.Status.Value()),
			slog.Float64("баллов", order.Accrual),
		)
		orders = append(orders, order)
	}
	if len(orders) == 0 {
		p.Warn("поиск заказов у пользователя. заказов нет.", slog.String("userID", userID.String()))
		return nil, storage.ErrOrdersNotFound
	}
	return orders, nil
}
