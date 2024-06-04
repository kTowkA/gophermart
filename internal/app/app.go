package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kTowkA/gophermart/internal/config"
	"github.com/kTowkA/gophermart/internal/storage"
)

type AppServer struct {
	storage storage.Storage
	config  config.Config
}

func NewAppServer(cfg config.Config, storage storage.Storage) (*AppServer, error) {
	app := AppServer{
		config:  cfg,
		storage: storage,
	}
	return &app, nil
}

func (a *AppServer) Start(ctx context.Context) error {
	r := chi.NewRouter()
	r.Use(checkRequestContentType)
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", a.rRegister)
		r.Post("/login", nil)
		r.Post("/orders", nil)
		r.Get("/orders", nil)
		r.Route("/balance", func(r chi.Router) {
			r.Get("/", nil)
			r.Post("/withdraw", nil)
		})
		r.Get("/withdrawals", nil)
	})
	if err := http.ListenAndServe(a.config.AddressApp, r); err != nil {
		return fmt.Errorf("запуск сервера. %w", err)
	}
	return nil
}
