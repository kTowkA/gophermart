package logger

import (
	"log/slog"
)

type Options struct {
	fileName string
	level    slog.Level
	useZap   bool
	textMode bool
}

type Option func(*Options)

func WithZap() Option {
	return func(o *Options) {
		o.useZap = true
	}
}
func WithTextMode() Option {
	return func(o *Options) {
		o.textMode = true
	}
}
func WithFile(filename string) Option {
	return func(o *Options) {
		o.fileName = filename
	}
}
func WithLevel(level slog.Level) Option {
	return func(o *Options) {
		o.level = level
	}
}
