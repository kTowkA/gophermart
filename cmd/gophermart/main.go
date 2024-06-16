package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kTowkA/gophermart/internal/app"
	"github.com/kTowkA/gophermart/internal/config"
	"github.com/kTowkA/gophermart/internal/logger"
)

func main() {
	// создание нашего логгера. выставляем уровень debug и бекенд от zap
	logger, err := logger.NewLog(logger.WithLevel(slog.LevelDebug), logger.WithZap())
	if err != nil {
		log.Fatal(err)
		return
	}
	defer logger.Close()

	// главный контекст приложения для отмены по ctrl+c + syscall.SIGTERM (он вроде отвечает за сигнал отмены в контейнерах)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// читаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("чтение конфигурационного файла", slog.String("ошибка", err.Error()))
		return
	}
	logger.Debug(
		"установленная конфигурация приложения",
		slog.String("адрес приложения для запуска", cfg.AddressApp()),
		slog.String("строка подключения базы данных", cfg.DatabaseURI()),
		slog.String("адрес расчета системы лояльности", cfg.AccruralSystemAddress()),
	)

	// запуск приложения с контекстом отмены по сигналу
	if err = app.RunApp(ctx, cfg, logger); err != nil {
		// если приложение было закрыто неправильно - выводим ошибку
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("запуск сервера", slog.String("ошибка", err.Error()))
		}
		return
	}
}
