package app

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kTowkA/gophermart/internal/config"
	"github.com/kTowkA/gophermart/internal/logger"
	"github.com/kTowkA/gophermart/internal/storage"
	"github.com/kTowkA/gophermart/internal/storage/postgres"
)

type AppServer struct {
	storage storage.Storage
	config  config.Config
	log     *slog.Logger
	server  *http.Server
}

func NewAppServer(cfg config.Config) *AppServer {
	app := AppServer{
		config: cfg,
	}
	return &app
}

func (a *AppServer) Start(ctx context.Context, log *logger.Log) error {
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

	a.server = &http.Server{
		Addr:    a.config.AddressApp,
		Handler: r,
	}
	a.log = log.WithGroup("application")
	if a.storage == nil {
		pst, err := postgres.New(ctx, a.config.DatabaseURI, log)
		if err != nil {
			return err
		}
		a.storage = pst
	}

	return a.server.ListenAndServe()
}

func (a *AppServer) Shutdown(ctx context.Context) error {
	a.storage.Close(ctx)
	return a.server.Shutdown(ctx)
}
func (a *AppServer) SetStorage(storage storage.Storage) {
	a.storage = storage
}
