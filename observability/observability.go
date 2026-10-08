package observability

import (
	"context"
	"errors"
	"evsched/config"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	EnvOTLPEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
	EnvOTLPHeaders  = "OTEL_EXPORTER_OTLP_HEADERS"
	EnvOTLPInsecure = "OTEL_EXPORTER_OTLP_INSECURE"
)

type Observability struct {
	TracerProvider *sdktrace.TracerProvider
	LogProvider    *sdklog.LoggerProvider
}

func New(ctx context.Context, cfg *config.Config) (*Observability, error) {
	rs, err := newResource(ctx, cfg)
	if err != nil {
		return nil, err
	}

	obs := &Observability{}

	var success bool
	defer func() {
		if !success {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = obs.Shutdown(shutdownCtx)
		}
	}()

	trace, err := NewTracerProvider(ctx, cfg, rs)
	if err != nil {
		return nil, fmt.Errorf("failed to init tracer provider: %w", err)
	}
	obs.TracerProvider = trace

	log, err := newLoggerProvider(ctx, rs, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init logger provider: %w", err)
	}
	obs.LogProvider = log

	success = true

	return obs, nil
}

func (o *Observability) Shutdown(ctx context.Context) error {
	var errs []error

	if o.TracerProvider != nil {
		if err := o.TracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown failed: %w", err))
		}
	}

	if o.LogProvider != nil {
		if err := o.LogProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("logger shutdown failed: %w", err))
		}
	}

	return errors.Join(errs...)
}

func newResource(ctx context.Context, cfg *config.Config) (*resource.Resource, error) {
	hostname, _ := os.Hostname()
	instanceID := uuid.NewString()[:8]
	if hostname != "" {
		instanceID = fmt.Sprintf("%s-%s", hostname, instanceID)
	}
	serviceName := fmt.Sprintf("%s-%s", cfg.Service.Name, cfg.Service.Environment)

	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(cfg.Service.Version),
			semconv.DeploymentEnvironment(cfg.Service.Environment),
			semconv.ServiceInstanceID(instanceID),
		),
	)
}

type OTLPConfig struct {
	Endpoint string
	Insecure bool
	Headers  map[string]string
}

// resolveOTLPConfig resolves exporter settings by prioritizing OS Environment variables
// over app config values
func resolveOTLPConfig(cfg *config.Config) OTLPConfig {

	endpoint := getEnvOrDefault(EnvOTLPEndpoint, cfg.OTel.Endpoint)

	insecure := cfg.OTel.Insecure
	if insecureStr := os.Getenv(EnvOTLPInsecure); insecureStr != "" {
		if parsed, err := strconv.ParseBool(insecureStr); err == nil {
			insecure = parsed
		}
	}

	var headers map[string]string
	if envHeaders := os.Getenv(EnvOTLPHeaders); envHeaders != "" {
		headers = parseHeaderString(envHeaders)
	} else if len(cfg.OTel.Headers) > 0 {
		headers = cfg.OTel.Headers
	}

	return OTLPConfig{
		Endpoint: endpoint,
		Insecure: insecure,
		Headers:  headers,
	}
}

func parseHeaderString(headerStr string) map[string]string {
	headers := make(map[string]string)
	pairs := strings.Split(headerStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return headers
}

func getEnvOrDefault(envKey, configValue string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	return configValue
}
