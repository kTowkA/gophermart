package app

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kTowkA/gophermart/internal/model"
	"github.com/kTowkA/gophermart/internal/storage"
)

func (a *AppServer) rBalance(w http.ResponseWriter, r *http.Request) {
	uc, ok := (r.Context().Value(userClaims{})).(UserClaims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	balance, err := a.storage.Balance(r.Context(), uc.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("content-type", "application/json")
	err = json.NewEncoder(w).Encode(balance)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
func (a *AppServer) rWithdraw(w http.ResponseWriter, r *http.Request) {
	if !checkContentType(r, []string{"application/json"}) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	req := model.RequestWithdraw{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	_, ok := validLuhnNumber(string(req.OrderNumber))
	if !ok {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	uc, ok := (r.Context().Value(userClaims{})).(UserClaims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = a.storage.Withdraw(r.Context(), uc.UserID, req)
	if errors.Is(err, storage.ErrWithdrawNotEnough) {
		w.WriteHeader(http.StatusPaymentRequired)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
func (a *AppServer) rWithdrawals(w http.ResponseWriter, r *http.Request) {
	uc, ok := (r.Context().Value(userClaims{})).(UserClaims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	withdrawals, err := a.storage.Withdrawals(r.Context(), uc.UserID)
	if errors.Is(err, storage.ErrWithdrawalsNotFound) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("content-type", "application/json")
	err = json.NewEncoder(w).Encode(withdrawals)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
