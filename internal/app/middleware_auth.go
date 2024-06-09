// middleware которая отвечает за то что все роуты кроме нескольких определенных сокрыты от неавторизованных пользователей
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

// чтобы передать информацию о пользователе дальше по запросам используем context.Value
// userClaims это структура для ключа context.Value чтобы избежать коллизий со стандартными типами
type userClaims struct{}

// middlewareAuthUser функция проверки на возможность доступа к методам API
func (a *AppServer) middlewareAuthUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// разрешаем доступ к некоторым ссылкам
		linksAllowedAllUsers := []string{
			"api/user/register",
			"api/user/login",
		}
		path := strings.Trim(r.URL.Path, "/")
		path = strings.ToLower(path)
		for _, l := range linksAllowedAllUsers {
			if strings.EqualFold(l, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// получаем токен из кук
		cookieToken, err := r.Cookie(cookieTokenName)
		if errors.Is(err, http.ErrNoCookie) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err != nil {
			a.log.Error("получение значения куки",
				slog.String("кука", cookieTokenName),
				slog.String("ошибка", err.Error()))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// получаем пользовательские данные
		uc, err := getUserClaimsFromToken(cookieToken.Value, a.config.Secret)
		if err != nil {
			a.log.Error("получение данных пользователя",
				slog.String("ошибка", err.Error()))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// добавляем в контекст данные пользователя
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userClaims{}, uc)))
	})
}
