package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kTowkA/gophermart/internal/config"
	"github.com/kTowkA/gophermart/internal/logger"
	"github.com/kTowkA/gophermart/internal/storage"
	"github.com/kTowkA/gophermart/internal/storage/postgres"
	"golang.org/x/sync/errgroup"
)

type AppServer struct {
	storage storage.Storage
	config  config.Config
	log     *slog.Logger
	server  *http.Server
}

func RunApp(ctx context.Context, cfg config.Config, log *logger.Log) error {
	app := AppServer{
		log:    log.WithGroup("application"),
		config: cfg,
	}
	app.server = &http.Server{
		Addr:    cfg.AddressApp,
		Handler: app.createRoute(),
	}
	if cfg.DatabaseURI != "" {
		err := postgres.Migration(cfg.DatabaseURI)
		if err != nil {
			app.log.Error("проведение миграций", slog.String("DatabaseURI", cfg.DatabaseURI), slog.String("ошибка", err.Error()))
			return err
		}
		storage, err := postgres.NewStorage(ctx, cfg.DatabaseURI, log)
		if err != nil {
			app.log.Error("подключение к БД", slog.String("DatabaseURI", cfg.DatabaseURI), slog.String("ошибка", err.Error()))
			return err
		}
		app.storage = storage
		defer app.storage.Close(context.Background())
	}
	group, ctxErr := errgroup.WithContext(ctx)
	group.Go(func() error {
		app.updaterStatus(ctx)
		return nil
	})
	group.Go(func() (err error) {
		defer func() {
			errRec := recover()
			if errRec != nil {
				err = fmt.Errorf("a panic occurred: %v", errRec)
			}
		}()
		if err := app.server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return fmt.Errorf("listen and server has failed: %w", err)
		}
		return nil
	})
	group.Go(func() error {
		// defer log.Print("server has been shutdown")
		<-ctxErr.Done()

		shutdownTimeoutCtx, cancelShutdownTimeoutCtx := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdownTimeoutCtx()
		if err := app.server.Shutdown(shutdownTimeoutCtx); err != nil {
			app.log.Error("остановка сервера", slog.String("ошибка", err.Error()))
			// log.Printf("an error occurred during server shutdown: %v", err)
		}
		return nil
	})
	return group.Wait()

}

func (a *AppServer) createRoute() http.Handler {
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
	return r
}
