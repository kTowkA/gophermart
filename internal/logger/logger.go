package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

// Log - это кастомный логгер с интерфейсом slog, но с возможностью установки бэкенда zap и сохранения в файл (в дополнении к стандартному выводу)
type Log struct {
	*slog.Logger
	file *os.File
	zl   *zap.Logger
}

// NewLog создает новый кастомный логгер с использованием опций настройки optins
func NewLog(optins ...Option) (*Log, error) {
	opt := new(Options)
	for _, o := range optins {
		o(opt)
	}

	// если уровень не передавался в опциях - используем info
	if opt.level == 0 {
		opt.level = slog.LevelInfo
	}

	// опция использования zap для бэкенда
	if opt.useZap {
		zc := zap.NewProductionConfig()
		zc.OutputPaths = []string{
			"stdout",
		}
		if opt.fileName != "" {
			zc.OutputPaths = append(zc.OutputPaths, opt.fileName)
		}
		zc.Encoding = "json"
		if opt.textMode {
			zc.Encoding = "console"
		}
		if opt.level != slog.LevelInfo {
			switch opt.level {
			case slog.LevelWarn:
				zc.Level.SetLevel(zap.WarnLevel)
			case slog.LevelDebug:
				zc.Level.SetLevel(zap.DebugLevel)
			case slog.LevelError:
				zc.Level.SetLevel(zap.ErrorLevel)
			}
		}
		l, err := zc.Build()
		if err != nil {
			return nil, fmt.Errorf("создание логера. сборка zap логера. %w", err)
		}
		return &Log{
			Logger: slog.New(zapslog.NewHandler(l.Core(), nil)),
			zl:     l,
		}, nil
	}

	// создание нашего логера на основе slog
	l := &Log{}

	// создаем источник для вывода лога, по умолчанию - стандартный
	var w io.Writer = os.Stdout

	// было указано сохранение в файл
	if opt.fileName != "" {
		file, err := os.OpenFile(opt.fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			return nil, fmt.Errorf("создание логера. открытие файла.  %w", err)
		}
		l.file = file
		// добавляем к стандартному источнику вывода вывод в файл
		w = io.MultiWriter(os.Stdout, file)
	}

	var h slog.Handler = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: opt.level})
	if opt.textMode {
		h = slog.NewTextHandler(w, &slog.HandlerOptions{Level: opt.level})
	}
	l.Logger = slog.New(h)

	return l, nil
}

// Close закрытие логера. Нужно при использовании zap и сохранении в файл
func (l *Log) Close() error {
	// если использовался файл - закрываем его
	if l.file != nil {
		return l.file.Close()
	}
	// если использовался zap - вызываем Sync. На ошибку не проверяем, там было открыто issue и разработчики советовали пропустить, на unix-системах возвращает не nil
	if l.zl != nil {
		_ = l.zl.Sync()
	}
	return nil
}
