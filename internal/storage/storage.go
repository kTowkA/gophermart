// пакет определяющий интерфейс хранилища
package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/kTowkA/gophermart/internal/model"
)

type Storage interface {
	// SaveUser сохраняет в хранилище пользователя с логином login и паролем(хешом от пароля) passwordHash
	// возвращает сгенерированный uuid (id пользователя) или ошибку
	// может вернуть ошибку ErrLoginIsUsed, если такой логин уже занят
	SaveUser(ctx context.Context, login, hashPassword string) (uuid.UUID, error)

	// UserID возвращает пользовательский id по переданному login
	// Если по такому логину не находит пользователя, то возвращает ErrUserNotFound
	UserID(ctx context.Context, login string) (uuid.UUID, error)

	// HashPassword по переданному id пользователя userID возвращает хранящийся хеш пароля из хранилища для сравнения.
	// Если по такому id не находит пользователя, то возвращает ErrUserNotFound
	HashPassword(ctx context.Context, userID uuid.UUID) (string, error)

	// SaveOrder сохраняет заказ orderNum в системе, привязывая его к пользователю userID.
	// Возвращает структуру ErrorWithHttpStatus с ошибкой бд и рекомендуемым кодом http.
	// Возвращает ErrOrderWasUploadByAnotherUser + http.StatusConflict если другой пользователь уже загрузил заказ с таким номером.
	// Возвращает ErrOrderWasAlreadyUpload+http.StatusOK если пользователь уже сохранял заказ order.
	// В других случаях при возникновении ошибки возвращает ошибку и http.StatusInternalServerError.
	// В случае принятия заказа ошибка будет nil и код http.StatusAccepted
	SaveOrder(ctx context.Context, userID uuid.UUID, orderNum model.OrderNumber) ErrorWithHTTPStatus

	// Orders возвращает все заказы для пользователя userID.
	// При отсутствии заказов возвращает ErrOrdersNotFound
	Orders(ctx context.Context, userID uuid.UUID) (model.ResponseOrders, error)

	// Balance возвращает информацию о балансе пользователя с id userID
	Balance(ctx context.Context, userID uuid.UUID) (model.ResponseBalance, error)

	// Withdrawals возвращает информацию о списаниях пользователя.
	// При отсутсвии истории списания возвращает ErrWithdrawalsNotFound
	Withdrawals(ctx context.Context, userID uuid.UUID) (model.ResponseWithdrawals, error)

	// Withdraw списывает баллы (RequestWithdraw.Sum) с накопительного счета на заказ requestWithdraw.OrderNumber.
	// При нехватке средств на балансе возвращает ErrWithdrawNotEnough
	Withdraw(ctx context.Context, userID uuid.UUID, requestWithdraw model.RequestWithdraw) error

	// OrdersByStatuses получает список из заказов у которых статус входит в заданную группу статусов statuses.
	// При отсутствии подходящих статусов возвращает ErrOrdersNotFound.
	// Для пагинации служат limit - максимальное количество данных для возврата и offset - смещение относительно начала подходящей выборки
	OrdersByStatuses(ctx context.Context, statuses []model.Status, limit, offset int) (model.ResponseOrders, error)

	// UpdateOrders обновляет информацию о заказе info.
	// Возвращает ErrNothingHasBeenDone если данные в репозитории уже актальны.
	// При отсутствии заказов с переданным номером возвращает ErrOrdersNotFound
	UpdateOrder(ctx context.Context, info model.ResponseAccuralSystem) error

	// UpdateOrders обновляет информацию о группе заказов info
	UpdateOrders(ctx context.Context, info []model.ResponseAccuralSystem) (int, error)

	// Close закрывает соединение с хранилищем
	Close(ctx context.Context) error
}
