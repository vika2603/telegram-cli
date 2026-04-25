// Package logging constructs zap loggers from config and redacts sensitive fields.
package logging

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/vika2603/telegram-cli/internal/config"
)

// closableFile is the minimum *os.File surface BuildLogger holds for the
// returned cleanup callback. An interface (not a concrete *os.File) so tests
// can wrap the real handle with a probe that counts Close() calls.
type closableFile interface {
	io.Writer
	Close() error
}

// openFileForLog is the log-file open seam. Tests replace it via
// withCloseProbe in logger_test.go; production uses defaultOpenFileForLog.
var openFileForLog = defaultOpenFileForLog

func defaultOpenFileForLog(path string, flag int, perm os.FileMode) (closableFile, error) {
	return os.OpenFile(path, flag, perm)
}

// BuildLogger returns the configured zap logger plus a cleanup callback that
// callers MUST invoke when done. The callback Syncs the logger and closes any
// log file opened by [log].file. A separate callback keeps file-handle
// ownership visible at call sites and avoids leaking the fd when the process
// is long-running.
func BuildLogger(c config.Config) (*zap.Logger, func(), error) {
	level := zapcore.WarnLevel
	if c.Log.Level != nil {
		_ = level.UnmarshalText([]byte(*c.Log.Level))
	}
	format := "console"
	if c.Log.Format != nil {
		format = *c.Log.Format
	}
	file := ""
	if c.Log.File != nil {
		file = *c.Log.File
	}

	enc := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	if format == "json" {
		enc = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	}

	sink := zapcore.Lock(os.Stderr)
	var openedFile closableFile
	if file != "" {
		f, err := openFileForLog(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return nil, func() {}, fmt.Errorf("open log file %s: %w", file, err)
		}
		openedFile = f
		sink = zapcore.AddSync(f)
	}
	core := zapcore.NewCore(enc, sink, level)
	logger := zap.New(&redactingCore{Core: core}, zap.WithCaller(false))
	cleanup := func() {
		_ = logger.Sync()
		if openedFile != nil {
			_ = openedFile.Close()
			openedFile = nil
		}
	}
	return logger, cleanup, nil
}

// redactedFieldKeys holds the zap field names whose values are rewritten to
// "[redacted]" before emission.
var redactedFieldKeys = map[string]struct{}{
	"api_hash": {},
	"code":     {},
	"password": {},
	"session":  {},
}

// redactingCore wraps a zapcore.Core so fields with sensitive names are
// replaced with a constant "[redacted]" string before the underlying core
// sees them. Both With() and Write() must filter.
type redactingCore struct {
	zapcore.Core
}

func (c *redactingCore) With(fields []zapcore.Field) zapcore.Core {
	return &redactingCore{Core: c.Core.With(redactFields(fields))}
}

func (c *redactingCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *redactingCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(ent, redactFields(fields))
}

func redactFields(in []zapcore.Field) []zapcore.Field {
	out := make([]zapcore.Field, len(in))
	for i, f := range in {
		if _, mask := redactedFieldKeys[f.Key]; mask {
			out[i] = zapcore.Field{Key: f.Key, Type: zapcore.StringType, String: "[redacted]"}
			continue
		}
		out[i] = f
	}
	return out
}
