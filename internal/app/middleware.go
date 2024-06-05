package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type userClaims string

func (a *AppServer) checkOnlyAuthUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.Trim(r.URL.Path, "/")
		if strings.EqualFold("api/user/register", path) || strings.EqualFold("api/user/login", path) {
			next.ServeHTTP(w, r)
			return
		}
		cookieToken, err := r.Cookie(cookieTokenName)
		if errors.Is(err, http.ErrNoCookie) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		uc, err := getUserClaimsFromToken(cookieToken.Value, a.config.Secret)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		modifyRequest := r.WithContext(context.WithValue(r.Context(), userClaims("claims"), uc))
		next.ServeHTTP(w, modifyRequest)
	})
}
func checkContentType(r *http.Request, allowedContentTypes []string) bool {
	ct := r.Header.Get("content-type")
	for _, act := range allowedContentTypes {
		if strings.HasPrefix(ct, act) {
			return true
		}
	}
	return false
}

// checkPOSTBody проверяем пост запрос, что есть тело
func checkPOSTBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		// проверка, что нам вообще что-то передали
		if r.Body == nil {
			http.Error(w, "не были переданы входные данные", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}
