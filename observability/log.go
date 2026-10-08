package observability

import (
	"context"
	"evsched/config"
	"fmt"
	"strings"

	"go.opentelemetry.io/contrib/processors/minsev"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

func newLoggerProvider(ctx context.Context, rs *resource.Resource, cfg *config.Config) (*sdklog.LoggerProvider, error) {

	if !cfg.OTel.Log.Enabled {
		return nil, nil
	}

	opts := buildLogExporterOptions(cfg)
	logExporter, err := otlploggrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create otlp log exporter: %w", err)
	}

	level := parseSeverity(cfg.Log.Level)
	processor := minsev.NewLogProcessor(
		sdklog.NewBatchProcessor(logExporter),
		&LogSeverity{level: level},
	)

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(rs),
		sdklog.WithProcessor(processor),
	)
	global.SetLoggerProvider(lp)

	return lp, nil

}

func buildLogExporterOptions(cfg *config.Config) []otlploggrpc.Option {
	var opts []otlploggrpc.Option
	otlpConfig := resolveOTLPConfig(cfg)

	if otlpConfig.Endpoint != "" {
		opts = append(opts, otlploggrpc.WithEndpoint(otlpConfig.Endpoint))
	}
	if otlpConfig.Insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(otlpConfig.Headers))
	}

	return opts
}

type LogSeverity struct {
	level log.Severity
}

func (l *LogSeverity) Severity() log.Severity {
	return l.level
}

func parseSeverity(level string) log.Severity {
	switch strings.ToLower(level) {
	case "trace":
		return log.SeverityTrace

	case "debug":
		return log.SeverityDebug

	case "info":
		return log.SeverityInfo

	case "warn":
		return log.SeverityWarn

	case "error":
		return log.SeverityError

	default:
		return log.SeverityInfo
	}
}
