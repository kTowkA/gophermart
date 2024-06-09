// в этом файле описаны методы регистрации, авторизации пользователя
package app

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/kTowkA/gophermart/internal/model"
	"github.com/kTowkA/gophermart/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// rRegister хендлер для регистрации пользователей
func (a *AppServer) rRegister(w http.ResponseWriter, r *http.Request) {
	if !checkContentType(r, []string{"application/json"}) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// раскодируем переданные данные
	req := model.RequestRegister{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		a.log.Error("декодирование запроса", slog.String("ошибка", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// генерируем хеш от пароля с помощью пакета bcrypt
	bytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		a.log.Error("генерация пароля", slog.String("логин", req.Login), slog.String("ошибка", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// сохраняем пользователя в хранилище
	userID, err := a.storage.SaveUser(r.Context(), req.Login, string(bytes))
	// если такой логин уже занят, то возвращаем конфликт
	if errors.Is(err, storage.ErrLoginIsUsed) {
		a.log.Info("сохранение пользователя. логин уже занят", slog.String("логин", req.Login))
		w.WriteHeader(http.StatusConflict)
		return
	}
	if err != nil {
		a.log.Error("сохранение пользователя", slog.String("логин", req.Login), slog.String("ошибка", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// генерируем новый токен
	token, err := buildJWTString(userID, req.Login, a.config.Secret, 24*time.Hour)
	if err != nil {
		a.log.Error("генерация токена", slog.String("логин", req.Login), slog.String("ошибка", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// выставляем новый токен в куках, чтобы пользователь дальше его продолжил использовать
	http.SetCookie(w, &http.Cookie{Name: cookieTokenName, Value: token})

	w.WriteHeader(http.StatusOK)
}

// rLogin хендлер для получения токена для работы
func (a *AppServer) rLogin(w http.ResponseWriter, r *http.Request) {
	if !checkContentType(r, []string{"application/json"}) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// раскодируем переданные данные
	req := model.RequestLogin{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		a.log.Error("декодирование запроса", slog.String("ошибка", err.Error()))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	userID, err := a.storage.UserID(r.Context(), req.Login)
	if errors.Is(err, storage.ErrUserNotFound) {
		a.log.Info("поиск пользователя. пользователь не найден", slog.String("логин", req.Login))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err != nil {
		a.log.Error("поиск пользователя.", slog.String("логин", req.Login), slog.String("ошибка", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hashPassword, err := a.storage.HashPassword(r.Context(), userID)
	if errors.Is(err, storage.ErrUserNotFound) {
		a.log.Info("получение хеша пароля. пользователь не найден", slog.String("логин", req.Login))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err != nil {
		a.log.Error("получение хеша пароля.", slog.String("логин", req.Login), slog.String("ошибка", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(req.Password))
	if err != nil {
		a.log.Error("сравнение пароля и сохраненного хеша", slog.String("логин", req.Login), slog.String("ошибка", err.Error()))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// генерируем новый токен
	token, err := buildJWTString(userID, req.Login, a.config.Secret, 24*time.Hour)
	if err != nil {
		a.log.Error("генерация токена", slog.String("логин", req.Login), slog.String("ошибка", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// выставляем новый токен в куках, чтобы пользователь дальше его продолжил использовать
	http.SetCookie(w, &http.Cookie{Name: cookieTokenName, Value: token})

	w.WriteHeader(http.StatusOK)
}
