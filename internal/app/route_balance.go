package app

import (
	"encoding/json"
	"net/http"
)

func (a *AppServer) rBalance(w http.ResponseWriter, r *http.Request) {
	uc, ok := (r.Context().Value(userClaims("claims"))).(UserClaims)
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
	// w.WriteHeader(http.StatusOK)
}
