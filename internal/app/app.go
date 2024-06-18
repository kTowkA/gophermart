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

// AppServer структура нашего приложения
type AppServer struct {
	// storage хранилище
	storage storage.Storage
	// config конфигурация
	config config.Config
	// log slog логгер
	log *slog.Logger
	// server http сервер
	server *http.Server
}

// RunApp запуск приложения
func RunApp(ctx context.Context, cfg config.Config, log *logger.Log) error {
	app := AppServer{
		log:    log.WithGroup("application"),
		config: cfg,
	}
	app.server = &http.Server{
		Addr:    cfg.AddressApp(),
		Handler: app.createRoute(),
	}

	if cfg.DatabaseURI() == "" {
		app.log.Error("невозможно запустить приложение. отсутствует строка подключения к базе данных")
	}

	storage, err := postgres.NewStorage(ctx, cfg.DatabaseURI(), log)
	if err != nil {
		app.log.Error("подключение к БД", slog.String("DatabaseURI", cfg.DatabaseURI()), slog.String("ошибка", err.Error()))
		return err
	}
	app.storage = storage
	// так как в pgx.Pool не используется контекст для закрытия обходимся context.Background()
	defer app.storage.Close(context.Background())

	group, ctxErr := errgroup.WithContext(ctx)

	group.Go(func() error {
		// наш обработчик для работы с накопительной системой
		app.updaterStatus(ctx)
		return nil
	})

	group.Go(func() (err error) {
		defer func() {
			errRec := recover()
			if errRec != nil {
				err = fmt.Errorf("при работе приложения произошла паника: %v", errRec)
			}
		}()

		// запускаем приложение
		if err := app.server.ListenAndServe(); err != nil {
			// если было штатное завершение - ничего не возвращаем
			if !errors.Is(err, http.ErrServerClosed) {
				app.log.Error("во время запуска или работы сервера произошла ошибка", slog.String("ошибка", err.Error()))
				return err
			}
		}
		return nil
	})
	group.Go(func() error {
		defer app.log.Info("работа приложения была завершена")

		// ждем контекст отмены
		<-ctxErr.Done()

		// пытаемся завершить приложение за wait времени
		wait := 10 * time.Second
		shutdownCtx, cancelShutdownCtx := context.WithTimeout(context.Background(), wait)
		defer cancelShutdownCtx()
		if err := app.server.Shutdown(shutdownCtx); err != nil {
			app.log.Error("остановка сервера", slog.String("ошибка", err.Error()))
		}
		return nil
	})
	return group.Wait()
}

// createRoute создание обработчика
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
