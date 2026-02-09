package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/alexduzi/labratelimiter/internal/config"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
	WithContext(ctx context.Context) Logger
}

type SlogLogger struct {
	logger *slog.Logger
}

func NewLogger(cfg *config.Config) Logger {
	var level slog.Level
	var handler slog.Handler

	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}

	var output io.Writer = os.Stdout

	switch cfg.LogFormat {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	default:
		handler = slog.NewTextHandler(output, opts)
	}

	// default fields
	logger := slog.New(handler).With(
		slog.String("service", cfg.AppName),
		slog.String("environment", cfg.AppEnv),
	)

	return &SlogLogger{
		logger: logger,
	}
}

func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *SlogLogger) With(args ...any) Logger {
	return &SlogLogger{
		logger: l.logger.With(args...),
	}
}

func (l *SlogLogger) WithContext(ctx context.Context) Logger {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return l.With(slog.String("request_id", requestID))
	}
	return l
}

func (l *SlogLogger) GetLogger() *slog.Logger {
	return l.logger
}
