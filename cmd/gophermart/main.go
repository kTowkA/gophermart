package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kTowkA/gophermart/internal/app"
	"github.com/kTowkA/gophermart/internal/config"
	"github.com/kTowkA/gophermart/internal/logger"
	"golang.org/x/sync/errgroup"
)

func main() {
	logger, err := logger.New(logger.WithLevel(slog.LevelDebug), logger.WithZap())
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	// корневой контекст приложения
	rootCtx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelCtx()

	g, ctx := errgroup.WithContext(rootCtx)

	// нештатное завершение программы по таймауту
	// происходит, если после завершения контекста
	// приложение не смогло завершиться за отведенный промежуток времени
	context.AfterFunc(ctx, func() {
		ctx, cancelCtx := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancelCtx()

		<-ctx.Done()
		logger.Error("failed to gracefully shutdown the service")
		os.Exit(1)
	})

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("чтение конфигурационного файла", slog.String("ошибка", err.Error()))
		return
	}

	logger.Debug("конфигурация", slog.String("Address App", cfg.AddressApp), slog.String("Database URI", cfg.DatabaseURI), slog.String("Accural System Address", cfg.AccuralSystemAddress))

	myapp := app.NewAppServer(cfg)
	// запуск сервера
	g.Go(func() (err error) {
		defer func() {
			errRec := recover()
			if errRec != nil {
				err = fmt.Errorf("a panic occurred: %v", errRec)
			}
		}()
		if err = myapp.Start(context.Background(), logger); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return fmt.Errorf("listen and server has failed: %w", err)
		}
		return nil
	})

	// отслеживаем успешное завершение работы сервера
	g.Go(func() error {
		defer logger.Info("server has been shutdown")
		<-ctx.Done()

		shutdownTimeoutCtx, cancelShutdownTimeoutCtx := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancelShutdownTimeoutCtx()
		if err := myapp.Shutdown(shutdownTimeoutCtx); err != nil {
			logger.Error("an error occurred during server shutdown", slog.String("ошибка", err.Error()))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error("", slog.String("", err.Error()))
	}

}
