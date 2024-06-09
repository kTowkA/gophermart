// отдельно вынес прокладку для хранения информации о запросах
package app

import (
	"log/slog"
	"net/http"
	"time"
)

// responseData дополнительная ифнормация из ответа
type responseData struct {
	// status http код
	status int
	// size размер ответа
	size int
}

// loggingResponseWriter наш писатель со встроенными дополнительными полями
type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// middlewareLog сама функция логирования запросов
func (a *AppServer) middlewareLog(h http.Handler) http.Handler {

	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   &responseData{},
		}

		h.ServeHTTP(&lw, r)

		duration := time.Since(start)
		a.log.Info(
			"запрос",
			slog.String("uri", r.RequestURI),
			slog.String("http метод", r.Method),
			slog.Duration("длительность", duration),
			slog.Int("статус", lw.responseData.status),
			slog.Int("размер ответа", lw.responseData.size),
		)
	}

	return http.HandlerFunc(logFn)
}
