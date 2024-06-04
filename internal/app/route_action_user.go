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
	// проверка, что нам вообще что-то передали
	if r.Body == nil {
		http.Error(w, "не были переданы входные данные", http.StatusBadRequest)
		return
	}
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
	token, err := buildJWTString(userID, a.config.Secret, 24*time.Hour)
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
}

// buildJWTString создаёт токен и возвращает его в виде строки.
func buildJWTString(userID uuid.UUID, secret string, dur time.Duration) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(dur)),
		},
		// собственное утверждение
		UserID: userID,
	})

	// создаём строку токена
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}
