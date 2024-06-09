package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kTowkA/gophermart/internal/config"
	"github.com/kTowkA/gophermart/internal/logger"
	"github.com/kTowkA/gophermart/internal/storage"
)

type AppServer struct {
	storage storage.Storage
	config  config.Config
	log     *slog.Logger
}

func NewAppServer(cfg config.Config, storage storage.Storage, log *logger.Log) (*AppServer, error) {
	app := AppServer{
		config:  cfg,
		storage: storage,
		log:     log.WithGroup("application"),
	}
	return &app, nil
}

func (a *AppServer) Start(ctx context.Context) error {
	r := chi.NewRouter()
	r.Use(middlewarePostBody, a.middlewareAuthUser, a.middlewareLog)
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", a.rRegister)
		r.Post("/login", a.rLogin)
		r.Post("/orders", a.rOrdersPost)
		r.Get("/orders", a.rOrdersGet)
		r.Route("/balance", func(r chi.Router) {
			r.Get("/", a.rBalance)
			r.Post("/withdraw", a.rWithdraw)
		})
		r.Get("/withdrawals", a.rWithdrawals)
	})
	if err := http.ListenAndServe(a.config.AddressApp, r); err != nil {
		return fmt.Errorf("запуск сервера. %w", err)
	}
	return nil
}
