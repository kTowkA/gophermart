// здесь содержатся вспомогательные методы проверки
package app

import (
	"net/http"
	"strings"
)

// checkContentType вначале было middleware, но теперь просто вспомогательная функция для проверки Сontent-type
func checkContentType(r *http.Request, allowedContentTypes []string) bool {
	ct := r.Header.Get("Сontent-type")
	for _, act := range allowedContentTypes {
		if strings.HasPrefix(ct, act) {
			return true
		}
	}
	return false
}

// middlewarePostBody проверяем пост запрос на то, что он имеет тело тело
func middlewarePostBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}
		// проверка, что нам вообще что-то передали
		if r.Body == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}
