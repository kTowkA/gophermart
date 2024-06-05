package app

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func checkRequestContentType(next http.Handler) http.Handler {
	var allowedTypes = []string{
		"application/json",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("content-Type")
		for _, a := range allowedTypes {
			if strings.HasPrefix(ct, a) {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.WriteHeader(http.StatusBadRequest)
	})
}
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
		_, err = getUserIDFromToken(cookieToken.Value, a.config.Secret)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// getUserIDFromToken - получает ID из JWT токена
func getUserIDFromToken(tokenString, secret string) (uuid.UUID, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("неожиданный метод подписи: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		})
	if err != nil {
		return uuid.UUID{}, err
	}

	if !token.Valid {
		return uuid.UUID{}, fmt.Errorf("токен не прошел проверку")
	}

	return claims.UserID, nil
}

// checkPOSTJSON проверяем пост запрос, что есть тело
func checkPOSTJSON(next http.Handler) http.Handler {
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
