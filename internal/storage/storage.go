package storage

import (
	"context"

	"github.com/google/uuid"
)

type Storage interface {
	StorageUser
}

type StorageUser interface {
	// SaveUser сохраняет в хранилище пользователя с логином login и паролем(хешом от пароля) passwordHash
	// возвращает сгенерированный uuid или ошибку
	// может вернуть ошибку ErrLoginIsUsed, если такой логин уже занят
	SaveUser(ctx context.Context, login, passwordHash string) (uuid.UUID, error)
	// PasswordHash по логину login возвращает уникальный uuid пользователя и хранящийся хеш пароля из хранилища для сравнения.
	// Если по такому логину не находит пользователя, то возвращает ErrUserNotFound
	PasswordHash(ctx context.Context, login string) (uuid.UUID, string, error)
	// Close закрывает соединение с хранилищем
	Close(ctx context.Context) error
}
