package app

import (
	"encoding/json"
	"errors"
	"io"
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
	if !checkContentType(r, []string{"application/json"}) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
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

func (a *AppServer) rOrdersPost(w http.ResponseWriter, r *http.Request) {
	if !checkContentType(r, []string{"text/plain"}) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	orderBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	order, ok := validLuhnNumber(string(orderBytes))
	if !ok {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	uc, ok := (r.Context().Value(userClaims("claims"))).(UserClaims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = a.storage.SaveOrder(r.Context(), uc.UserID, order)
	switch {
	case errors.Is(err, storage.ErrOrderWasAlreadyUpload):
		w.WriteHeader(http.StatusOK)
	case errors.Is(err, storage.ErrOrderWasUploadByAnotherUser):
		w.WriteHeader(http.StatusConflict)
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusCreated)
	}
}
func (a *AppServer) rOrdersGet(w http.ResponseWriter, r *http.Request) {

	uc, ok := (r.Context().Value(userClaims("claims"))).(UserClaims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	orders, err := a.storage.Orders(r.Context(), uc.UserID)
	if errors.Is(err, storage.ErrOrdersNotFound) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// log.Println("ORDERS", orders)
	// ordersB, err := json.Marshal(orders)
	w.Header().Add("content-type", "application/json")
	err = json.NewEncoder(w).Encode(orders)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// w.Write(ordersB)
	w.WriteHeader(http.StatusOK)
}
