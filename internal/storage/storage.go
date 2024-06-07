// пакет определяющий интерфейс хранилища
// реализуем, что пользователь определяется по его ID (представлен uuid), для предполагаемой возможной смены логина
package storage

import (
	"context"

	"github.com/google/uuid"
	"github.com/kTowkA/gophermart/internal/model"
)

type Storage interface {
	StorageUser
}

type StorageUser interface {
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
	// SaveOrder сохраняет заказ order в системе, привязывая его к пользователю userID
	// может вернуть ErrOrderWasUploadByAnotherUser если другой пользователь уже загрузил заказ с таким номером
	// или ErrOrderWasAlreadyUpload если пользователь уже сохранял заказ order
	SaveOrder(ctx context.Context, userID uuid.UUID, order int64) error
	// Orders возвращает все заказы для пользователя userID
	// при отсутствии заказов возвращает ErrOrdersNotFound
	Orders(ctx context.Context, userID uuid.UUID) (model.ResponseOrders, error)
	// Balance возвращает информацию о балансе пользователя с id userID
	Balance(ctx context.Context, userID uuid.UUID) (model.ResponseBalance, error)
	// Close закрывает соединение с хранилищем
	Close(ctx context.Context) error
}
