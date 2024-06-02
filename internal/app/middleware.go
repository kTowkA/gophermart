package app

import (
	"net/http"
	"strings"
)

func checkRequestContentType(next http.Handler) http.Handler {
	var allowedTypes = []string{
		"application/json",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok := false
		ct := r.Header.Get("content-Type")
		for _, a := range allowedTypes {
			if strings.HasPrefix(ct, a) {
				ok = true
				break
			}
		}
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}
