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

// updaterStatus это конвейер для взаимодействия с внешней системой расчета баллов лояльности. На первом шаге получаем заказы с неокончательными статусами и записываем их в канал, далее читаем эти заказы из канала и делаем обращение к внешней системе расчета баллов лояльности, результат записываем в канал и далее читаем из этого канала и обновляем в нашей базе данных соответсвующие заказы
func (a *AppServer) updaterStatus(ctx context.Context) {
	a.updateOrders(
		ctx,
		a.gettingInfoFromAccuralSystem(
			ctx,
			a.gettingOrders(ctx),
		),
	)
}

// updateOrders обновляет заказы обработанные системой рачета баллов лояльности
func (a *AppServer) updateOrders(ctx context.Context, accuralInfo <-chan model.ResponseAccuralSystem) {
	for ai := range accuralInfo {
		select {
		case <-ctx.Done():
			a.log.Debug("получен сигнал остановки. Выходим из функции обновления заказов")
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
			a.log.Info(
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

// updateOrdersGroup обновляет сразу группу. Можно использовать эту функцию вместо updateOrders, единственное - при тикере больше чем время получения новых  заказов с определенными статусами, могут быть дубли
func (a *AppServer) updateOrdersGroup(ctx context.Context, accuralInfo <-chan model.ResponseAccuralSystem) {
	toRecord := make([]model.ResponseAccuralSystem, 0, 100)
	ticker := time.NewTicker(time.Duration(a.config.UpdateGroupStatusesSec()) * time.Second)
	for {
		select {
		case <-ctx.Done():
			a.log.Debug("получен сигнал остановки. Выходим из функции обновления группы заказов")
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

// gettingInfoFromAccuralSystem запрос к внешней системе расчета баллов лояльности
func (a *AppServer) gettingInfoFromAccuralSystem(ctx context.Context, orders <-chan model.ResponseOrder) chan model.ResponseAccuralSystem {
	accuralInfo := make(chan model.ResponseAccuralSystem, 100)
	go func(ctx context.Context, orders <-chan model.ResponseOrder) {
		defer close(accuralInfo)
		req := resty.
			New().
			AddRetryCondition(func(r *resty.Response, err error) bool {
				return err != nil || r.StatusCode() == http.StatusTooManyRequests
			}).
			SetRetryCount(3).
			SetRetryWaitTime(5 * time.Second).
			SetBaseURL(a.config.AccruralSystemAddress()).
			R()
		for {
			select {
			case <-ctx.Done():
				a.log.Debug("получен сигнал остановки. Выходим из функции запросов к внешней системе расчета баллов лояльности")
				return
			case o := <-orders:
				a.log.Debug("получили новый заказ для проверки расчета баллов", slog.String("номер заказа", string(o.OrderNumber)), slog.String("статус", o.Status.Value()))
				result := model.ResponseAccuralSystem{}
				resp, err := req.SetContext(ctx).SetResult(&result).Get("/api/orders/" + string(o.OrderNumber))
				if err != nil {
					a.log.Error(
						"обращение к системе расчета баллов",
						slog.String("BaseURL", a.config.AccruralSystemAddress()),
						slog.String("path", "/api/orders/"+string(o.OrderNumber)),
						slog.String("ошибка", err.Error()),
					)
					continue
				}
				a.log.Debug(
					"результат обращения к системе расчета баллов лояльности",
					slog.Int("статус", resp.StatusCode()),
					slog.String("заказ", string(o.OrderNumber)),
					slog.Any("result", result),
				)
				switch resp.StatusCode() {
				case http.StatusOK:
					accuralInfo <- result
				case http.StatusNoContent:
					a.log.Info("система расчета баллов лояльности вернула статус, что заказ не зарегистрирован", slog.String("заказ", string(o.OrderNumber)))
				default:
					a.log.Info("система расчета баллов лояльности вернула код ошибки", slog.String("код", resp.Status()))
				}
			}
		}
	}(ctx, orders)
	return accuralInfo
}

// gettingOrders используется для получения заказов с определенными статусами
func (a *AppServer) gettingOrders(ctx context.Context) chan model.ResponseOrder {
	ordersCh := make(chan model.ResponseOrder, 100)
	// максимум заказов за запрос
	limit := 100
	// начальное смещение 0
	offset := 0
	go func() {
		defer close(ordersCh)

		// заказы с этими статусами будут проверяться во внешней систему расчета баллов лояльности
		wantSt := []model.Status{storage.StatusUndefined, storage.StatusNew, storage.StatusProcessing, storage.StatusRegistered}

		// это чисто для логов
		stVal := make([]string, len(wantSt))
		for i := range wantSt {
			stVal[i] = wantSt[i].Value()
		}

		for {
			select {
			case <-ctx.Done():
				a.log.Debug("получен сигнал остановки. Выходим из функции получения заказов с определенными статусами")
				return
			default:
			}

			// получаем заказы
			orders, err := a.storage.OrdersByStatuses(ctx, wantSt, limit, offset)
			if err != nil && !errors.Is(err, storage.ErrOrdersNotFound) {
				a.log.Error(
					"запрос заказов",
					slog.Any("статусы по запросу", stVal),
					slog.Int("лимит", limit),
					slog.Int("смещение", offset),
					slog.String("ошибка", err.Error()))
				continue
			}
			a.log.Debug("запрос заказов", slog.Any("всего заказов", len(orders)), slog.Any("статусы по запросу", stVal), slog.Int("лимит", limit), slog.Int("смещение", offset))
			if errors.Is(err, storage.ErrOrdersNotFound) {
				select {
				case <-ctx.Done():
					a.log.Debug("получен сигнал остановки. Выходим из функции получения заказов с определенными статусами")
					return
				case <-time.After(5 * time.Second): // ждем перед новой попыткой запроса
				}
				// сбрасываем смещение
				offset = 0
				continue
			}

			// заказы были найдены
			for _, o := range orders {
				a.log.Debug("подходящий заказ был найден", slog.String("номер заказа", string(o.OrderNumber)), slog.String("статус", o.Status.Value()))
				ordersCh <- o
			}

			if len(orders) < limit {
				select {
				case <-ctx.Done():
					a.log.Debug("получен сигнал остановки. Выходим из функции получения заказов с определенными статусами")
					return
				case <-time.After(5 * time.Second): // ждем перед новой попыткой запроса
				}
				continue
			}

			offset += limit
		}
	}()
	return ordersCh
}
