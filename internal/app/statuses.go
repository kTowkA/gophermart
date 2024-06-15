// файл где содержится логика работы со статусами заказа
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/kTowkA/gophermart/internal/model"
	"github.com/kTowkA/gophermart/internal/storage"
)

type Status string

func (a *AppServer) updaterStatus(ctx context.Context) {
	a.updateOrdersGroup(
		ctx,
		a.gettingInfoFromAccuralSystem(
			ctx,
			a.gettingOrders(ctx),
		),
	)
}
func (a *AppServer) updateOrders(ctx context.Context, accuralInfo <-chan model.ResponseAccuralSystem) {
	for ai := range accuralInfo {
		select {
		case <-ctx.Done():
			a.log.Debug("выход из updateOrders")
			return
		default:
		}
		err := a.storage.UpdateOrder(ctx, ai)
		switch {
		case errors.Is(err, storage.ErrOrdersNotFound):
			a.log.Info(
				"заказ не найден",
				slog.String("заказ", string(ai.OrderNumber)),
			)
		case errors.Is(err, storage.ErrNothingHasBeenDone):
			a.log.Debug(
				"попытка повторного обновления",
				slog.String("заказ", string(ai.OrderNumber)),
				slog.Float64("начислено баллов", ai.Accrual),
				slog.String("статус", string(ai.Status.Value())),
			)
		case err != nil:
			a.log.Error(
				"обновление заказа",
				slog.String("заказ", string(ai.OrderNumber)),
				slog.Float64("начислено баллов", ai.Accrual),
				slog.String("статус", string(ai.Status.Value())),
				slog.String("ошибка", err.Error()),
			)
		default:
			a.log.Debug(
				"заказ обновлен",
				slog.String("заказ", string(ai.OrderNumber)),
				slog.Float64("начислено баллов", ai.Accrual),
				slog.String("статус", string(ai.Status.Value())),
			)
		}
	}
}
func (a *AppServer) updateOrdersGroup(ctx context.Context, accuralInfo <-chan model.ResponseAccuralSystem) {
	toRecord := make([]model.ResponseAccuralSystem, 0, 100)
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			a.log.Debug("выход из updateOrdersGroup")
			return
		case ai := <-accuralInfo:
			toRecord = append(toRecord, ai)
			continue
		case <-ticker.C:
			if len(toRecord) == 0 {
				continue
			}
		}
		success, err := a.storage.UpdateOrders(ctx, toRecord)
		a.log.Debug("сохранение группы заказов", slog.Int("сохранено успешно", success), slog.Int("всего", len(toRecord)))
		if err != nil {
			a.log.Error("сохранение группы заказов", slog.String("ошибка", err.Error()))
			continue
		}
		toRecord = make([]model.ResponseAccuralSystem, 0, 100)
	}
}
func (a *AppServer) gettingInfoFromAccuralSystem(ctx context.Context, orders <-chan model.ResponseOrder) chan model.ResponseAccuralSystem {
	var accuralInfo chan model.ResponseAccuralSystem
	go func(ctx context.Context, orders <-chan model.ResponseOrder) {
		accuralInfo = make(chan model.ResponseAccuralSystem, 100)
		defer close(accuralInfo)

		req := resty.
			New().
			SetBaseURL(a.config.AccruralSystemAddress).
			R()
		for {
			select {
			case <-ctx.Done():
				a.log.Debug("выход из gettingInfoFromAccuralSystem")
				return
			case o := <-orders:
				result := model.ResponseAccuralSystem{}
				resp, err := req.SetContext(ctx).SetResult(&result).Get("/api/orders/" + string(o.OrderNumber))
				if err != nil {
					a.log.Error(
						"обращение к системе расчета баллов",
						slog.String("BaseURL", a.config.AccruralSystemAddress),
						slog.String("path", "/api/orders/"+string(o.OrderNumber)),
						slog.String("ошибка", err.Error()),
					)
					continue
				}
				switch resp.StatusCode() {
				case http.StatusOK:
					accuralInfo <- result
				case http.StatusInternalServerError:
					a.log.Info("система расчета баллов вернула код ошибки", slog.String("код", resp.Status()))
				case http.StatusNoContent:
					a.log.Info("система расчета баллов. заказа не зарегистрирован", slog.String("заказ", string(o.OrderNumber)))
				case http.StatusTooManyRequests:
					select {
					case <-ctx.Done():
						a.log.Debug("выход из gettingInfoFromAccuralSystem")
					case <-time.After(5 * time.Second):
					}
				}
			}
		}
	}(ctx, orders)
	return accuralInfo
}
func (a *AppServer) gettingOrders(ctx context.Context) chan model.ResponseOrder {
	var ordersCh chan model.ResponseOrder
	limit := 100
	offset := 0
	go func() {
		ordersCh = make(chan model.ResponseOrder, 100)
		wantSt := []model.Status{storage.StatusUndefined, storage.StatusNew, storage.StatusProcessing}
		defer close(ordersCh)
		for {
			select {
			case <-ctx.Done():
				a.log.Debug("выход из gettingOrders")
				return
			default:
			}
			orders, err := a.storage.OrdersByStatuses(ctx, wantSt, limit, offset)
			if errors.Is(err, storage.ErrOrdersNotFound) {
				select {
				case <-ctx.Done():
					a.log.Debug("выход из gettingOrders")
					return
				case <-time.After(5 * time.Second):
				}
				offset = 0
				continue
			}
			if err != nil {
				a.log.Error("получение заказов по типу статуса", slog.Any("статусы", wantSt), slog.Int("лимит", limit), slog.Int("смещение", limit))
				continue
			}
			for _, o := range orders {
				ordersCh <- o
			}
			offset += limit
		}
	}()
	return ordersCh
}
