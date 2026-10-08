package logging

import (
	"evsched/config"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"golang.org/x/net/context"
)

const (
	FieldRequestID = "request_id"
	FieldTraceID   = "trace_id"
	FieldSpanID    = "span_id"

	FieldMethod = "method"
	FieldPath   = "path"
)

func New(cfg *config.Config, provider *sdklog.LoggerProvider) *slog.Logger {
	var logger *slog.Logger

	if cfg.OTel.Log.Enabled && provider != nil {
		logger = otelslog.NewLogger(
			fmt.Sprintf("%s-%s", cfg.Service.Name, cfg.Service.Environment),
			otelslog.WithLoggerProvider(provider),
			otelslog.WithSource(true),
		)
	} else {
		// Defaults to stdout logger
		logger = initStdoutLogger(cfg.Log, cfg.Log.Format)
	}

	slog.SetDefault(logger)
	return logger
}

func initStdoutLogger(cfg config.LogConfig, format string) *slog.Logger {
	level := parseLevel(cfg.Level)
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if source, ok := a.Value.Any().(*slog.Source); ok && source != nil {
					source.File = filepath.Base(source.File)
				}
			}
			return a
		},
	}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

type contextKey struct{}

// FromContext extracts the logger stored in context, falling back to slog.Default().
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// BindContext stores the logger in context.
func BindContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug

	case "warn":
		return slog.LevelWarn

	case "errors":
		return slog.LevelError

	default:
		return slog.LevelInfo
	}
}
