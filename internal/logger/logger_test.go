package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogger(t *testing.T) {
	l, err := NewLog()
	require.NoError(t, err)
	err = l.Close()
	require.NoError(t, err)
	l, err = NewLog(WithLevel(slog.LevelError), WithTextMode())
	require.NoError(t, err)
	err = l.Close()
	require.NoError(t, err)
	l, err = NewLog(WithLevel(slog.LevelWarn), WithFile("test.log"))
	require.NoError(t, err)
	err = l.Close()
	require.NoError(t, err)
	l, err = NewLog(WithLevel(slog.LevelWarn), WithZap())
	require.NoError(t, err)
	err = l.Close()
	require.NoError(t, err)
	l, err = NewLog(WithLevel(slog.LevelWarn), WithZap(), WithFile("zap.log"))
	require.NoError(t, err)
	err = l.Close()
	require.NoError(t, err)
}
