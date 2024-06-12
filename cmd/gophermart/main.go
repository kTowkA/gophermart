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
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("чтение конфигурационного файла", slog.String("ошибка", err.Error()))
		return
	}
	myapp := app.NewAppServer(cfg, nil, logger)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	err = myapp.Start(ctx)
	if err != nil {
		logger.Error("работа сервера", slog.String("ошибка", err.Error()))
		return
	}
}
