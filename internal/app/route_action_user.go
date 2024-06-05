package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/kTowkA/gophermart/internal/model"
	"github.com/kTowkA/gophermart/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

var cookieTokenName = "app_token"

// rRegister хендлер для регистрации пользователей
func (a *AppServer) rRegister(w http.ResponseWriter, r *http.Request) {
	// раскодируем переданные данные
	req := model.RequestRegister{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "невалидные данные в запросе", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	// генерируем хеш от пароля с помощью пакета bcrypt
	bytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// сохраняем пользователя в хранилище
	userID, err := a.storage.SaveUser(r.Context(), req.Login, string(bytes))
	// если такой логин уже занят, то возвращаем конфликт
	if errors.Is(err, storage.ErrLoginIsUsed) {
		w.WriteHeader(http.StatusConflict)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// генерируем новый токен
	token, err := buildJWTString(userID, req.Login, a.config.Secret, 24*time.Hour)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// выставляем новый токен в куках, чтобы пользователь дальше его продолжил использовать
	http.SetCookie(w, &http.Cookie{Name: cookieTokenName, Value: token})

	w.WriteHeader(http.StatusOK)
}

// rLogin хендлер для получения токена для работы
func (a *AppServer) rLogin(w http.ResponseWriter, r *http.Request) {
	// раскодируем переданные данные
	req := model.RequestLogin{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "невалидные данные в запросе", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	userID, hashPassword, err := a.storage.PasswordHash(r.Context(), req.Login)
	if errors.Is(err, storage.ErrUserNotFound) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(req.Password))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// генерируем новый токен
	token, err := buildJWTString(userID, req.Login, a.config.Secret, 24*time.Hour)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// выставляем новый токен в куках, чтобы пользователь дальше его продолжил использовать
	http.SetCookie(w, &http.Cookie{Name: cookieTokenName, Value: token})

	w.WriteHeader(http.StatusOK)
}

// Claims — структура утверждений, которая включает стандартные утверждения и
// одно пользовательское UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID
	Login  string
}

// buildJWTString создаёт токен и возвращает его в виде строки.
func buildJWTString(userID uuid.UUID, userLogin, appSecret string, dur time.Duration) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(dur)),
		},
		// собственное утверждение
		UserID: userID,
		Login:  userLogin,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(appSecret))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}
