package app

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/kTowkA/gophermart/internal/storage"
)

type AppServer struct {
	storage storage.Storage
}

func NewAppServer() (*AppServer, error) {
	return nil, nil
}

func (a *AppServer) Start(ctx context.Context) error {
	r := chi.NewRouter()
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", nil)
		r.Post("/login", nil)
		r.Post("/orders", nil)
		r.Get("/orders", nil)
		r.Route("/balance", func(r chi.Router) {
			r.Get("/", nil)
			r.Post("/withdraw", nil)
		})
		r.Get("/withdrawals", nil)
	})
	return nil
}
