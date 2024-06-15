package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kTowkA/gophermart/internal/app"
	"github.com/kTowkA/gophermart/internal/config"
	"github.com/kTowkA/gophermart/internal/logger"
)

func main() {
	logger, err := logger.New(logger.WithLevel(slog.LevelDebug), logger.WithZap())
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Close()

	rootCtx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelCtx()

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("чтение конфигурационного файла", slog.String("ошибка", err.Error()))
		return
	}
	logger.Debug("конфигурация", slog.String("Address App", cfg.AddressApp), slog.String("Database URI", cfg.DatabaseURI), slog.String("Accural System Address", cfg.AccuralSystemAddress))

	if err = app.RunApp(rootCtx, cfg, logger); err != nil {
		log.Fatal(err)
	}

}
