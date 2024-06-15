package app

import (
	"context"
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
	server  *http.Server
}

func NewAppServer(cfg config.Config, storage storage.Storage, log *logger.Log) *AppServer {
	app := AppServer{
		config:  cfg,
		storage: storage,
		log:     log.WithGroup("application"),
	}
	return &app
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

	a.server = &http.Server{
		Addr:    a.config.AddressApp,
		Handler: r,
	}
	return a.server.ListenAndServe()
	// log.Println("!!!", a.server.ListenAndServe())
	// gr, gCtx := errgroup.WithContext(ctx)
	// gr.Go(func() error {
	// 	return a.server.ListenAndServe()
	// })
	// gr.Go(func() error {
	// 	<-gCtx.Done()
	// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// 	defer cancel()
	// 	return a.server.Shutdown(ctx)
	// })
	// gr.Go(func() error {
	// 	a.updaterStatus(ctx)
	// 	return nil
	// })
	// err := gr.Wait()
	// if err != nil {
	// 	a.log.Error("сервер", slog.String("ошибка", err.Error()))
	// }
	// return nil
}

func (a *AppServer) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}
