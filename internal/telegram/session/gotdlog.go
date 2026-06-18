package session

import (
	"context"

	gotdlog "github.com/gotd/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// gotdLogger adapts our *zap.Logger to gotd's log.Logger interface. gotd
// migrated telegram.Options.Logger from *zap.Logger to its own minimal
// slog-style interface, so the zap logger no longer fits directly.
func gotdLogger(l *zap.Logger) gotdlog.Logger {
	if l == nil {
		return gotdlog.Nop
	}
	return zapGotdLogger{l: l}
}

type zapGotdLogger struct{ l *zap.Logger }

func (z zapGotdLogger) Enabled(_ context.Context, level gotdlog.Level) bool {
	return z.l.Core().Enabled(zapLevel(level))
}

func (z zapGotdLogger) Log(_ context.Context, level gotdlog.Level, msg string, attrs ...gotdlog.Attr) {
	ce := z.l.Check(zapLevel(level), msg)
	if ce == nil {
		return
	}
	fields := make([]zap.Field, len(attrs))
	for i, a := range attrs {
		fields[i] = zap.Any(a.Key, a.Value.Any())
	}
	ce.Write(fields...)
}

// zapLevel maps gotd's slog-style levels (Debug=-4, Info=0, Warn=4, Error=8)
// onto zap levels.
func zapLevel(l gotdlog.Level) zapcore.Level {
	switch {
	case l >= gotdlog.LevelError:
		return zapcore.ErrorLevel
	case l >= gotdlog.LevelWarn:
		return zapcore.WarnLevel
	case l >= gotdlog.LevelInfo:
		return zapcore.InfoLevel
	default:
		return zapcore.DebugLevel
	}
}
