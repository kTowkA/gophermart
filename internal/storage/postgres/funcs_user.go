package postgres

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kTowkA/gophermart/internal/storage"
)

func (p *PStorage) SaveUser(ctx context.Context, login, hashPassword string) (uuid.UUID, error) {
	_, err := p.UserID(ctx, login)
	if err == nil {
		p.Warn("запрос на поиск пользователя по логину. логин занят", slog.String("логин", login))
		return uuid.UUID{}, storage.ErrLoginIsUsed
	}
	if !errors.Is(err, storage.ErrUserNotFound) {
		p.Error("запрос на поиск пользователя по логину", slog.String("логин", login), slog.String("ошибка", err.Error()))
		return uuid.UUID{}, err
	}
	userID := uuid.New()
	_, err = p.Exec(
		ctx,
		"INSERT INTO users(user_id,login,password_hash,adding_at) VALUES($1,$2,$3,$4)",
		userID,
		login,
		hashPassword, time.Now(),
	)
	if err != nil {
		p.Error("запрос на сохранение пользователя", slog.String("логин", login), slog.String("ошибка", err.Error()))
		return uuid.UUID{}, err
	}
	return userID, nil
}

func (p *PStorage) UserID(ctx context.Context, login string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := p.QueryRow(ctx, "SELECT user_id FROM users WHERE login=$1", login).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		p.Warn("запрос поиска ID пользователя по логину. пользователь не найден", slog.String("логин", login))
		return uuid.UUID{}, storage.ErrUserNotFound
	}
	if err != nil {
		p.Warn("запрос поиска ID пользователя по логину", slog.String("логин", login), slog.String("ошибка", err.Error()))
		return uuid.UUID{}, err
	}
	return userID, nil
}

func (p *PStorage) HashPassword(ctx context.Context, userID uuid.UUID) (string, error) {
	var hash string
	err := p.QueryRow(ctx, "SELECT password_hash FROM users WHERE user_id=$1", userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		p.Warn("запрос хеша пароля пользователя по userID. пользователь не найден", slog.String("userID", userID.String()))
		return "", storage.ErrUserNotFound
	}
	if err != nil {
		p.Warn("запрос хеша пароля пользователя по userID", slog.String("userID", userID.String()), slog.String("ошибка", err.Error()))
		return "", err
	}
	return hash, nil
}
