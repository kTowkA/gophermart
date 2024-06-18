package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/kTowkA/gophermart/internal/luhn"
	"github.com/kTowkA/gophermart/internal/model"
	"github.com/kTowkA/gophermart/internal/storage"
)

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

	_, ok := luhn.ValidateLuhnNumber(string(orderBytes))
	if !ok {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	uc, ok := (r.Context().Value(userClaims{})).(UserClaims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	orderErr := a.storage.SaveOrder(r.Context(), uc.UserID, model.OrderNumber(orderBytes))
	w.WriteHeader(orderErr.HttpStatus)
}
func (a *AppServer) rOrdersGet(w http.ResponseWriter, r *http.Request) {
	uc, ok := (r.Context().Value(userClaims{})).(UserClaims)
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
	w.Header().Add("content-type", "application/json")
	err = json.NewEncoder(w).Encode(orders)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
