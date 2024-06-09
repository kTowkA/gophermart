package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

type Log struct {
	*slog.Logger
	file *os.File
	zl   *zap.Logger
}

func New(optins ...Option) (*Log, error) {
	opt := new(Options)
	for _, o := range optins {
		o(opt)
	}
	if opt.level == 0 {
		opt.level = slog.LevelInfo
	}

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
	l := &Log{}
	var w io.Writer
	w = os.Stdout
	if opt.fileName != "" {
		file, err := os.OpenFile(opt.fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			return nil, fmt.Errorf("создание логера. открытие файла.  %w", err)
		}
		l.file = file
		w = io.MultiWriter(os.Stdout, file)
	}
	var h slog.Handler
	h = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: opt.level})
	if opt.textMode {
		h = slog.NewTextHandler(w, &slog.HandlerOptions{Level: opt.level})
	}
	l.Logger = slog.New(h)

	return l, nil
}
func (l *Log) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	if l.zl != nil {
		_ = l.zl.Sync()
	}
	return nil
}
