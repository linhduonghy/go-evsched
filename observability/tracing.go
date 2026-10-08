package observability

import (
	"context"
	"evsched/config"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func NewTracerProvider(ctx context.Context, cfg *config.Config, rs *resource.Resource) (*sdktrace.TracerProvider, error) {

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	if !cfg.OTel.Trace.Enabled {
		tp := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
		return tp, nil
	}

	opts := buildTraceExporterOptions(cfg)
	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create otlp trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(rs),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(1.0))),
	)

	otel.SetTracerProvider(tp)

	return tp, nil
}

func buildTraceExporterOptions(cfg *config.Config) []otlptracegrpc.Option {
	var opts []otlptracegrpc.Option
	otlpConfig := resolveOTLPConfig(cfg)

	if otlpConfig.Endpoint != "" {
		opts = append(opts, otlptracegrpc.WithEndpoint(otlpConfig.Endpoint))
	}
	if otlpConfig.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	if len(otlpConfig.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(otlpConfig.Headers))
	}

	return opts
}
