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
	logger, err := logger.New(logger.WithLevel(slog.LevelDebug))
	if err != nil {
		log.Fatal(err)
		return
	}
	defer logger.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("чтение конфигурационного файла", slog.String("ошибка", err.Error()))
		return
	}
	logger.Debug("конфигурация", slog.String("Address App", cfg.AddressApp), slog.String("Database URI", cfg.DatabaseURI), slog.String("Accural System Address", cfg.AccruralSystemAddress))
	if err = app.RunApp(ctx, cfg, logger); err != nil {
		logger.Error("запуск сервера", slog.String("ошибка", err.Error()))
		return
	}
}
