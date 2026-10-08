package observability

import (
	"evsched/config"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/log"
)

func newTestConfig() *config.Config {

	return &config.Config{
		Service: config.Service{
			Name:        "evsched",
			Version:     "1.0.0",
			Environment: "unit-test",
		},
		Log: config.LogConfig{
			Level:  "debug",
			Format: "text",
		},
		OTel: config.OTelConfig{
			Endpoint: "localhost:4317",
			Insecure: true,
			Headers: map[string]string{
				"x-api-key": "test-key",
			},
			Trace: config.OtelTraceConfig{Enabled: false},
			Log:   config.OtelLogConfig{Enabled: false},
		},
	}
}

func TestResolveOTLPConfig_EnvPriority(t *testing.T) {
	os.Setenv(EnvOTLPEndpoint, "env-collector:4317")
	os.Setenv(EnvOTLPInsecure, "true")
	os.Setenv(EnvOTLPHeaders, "envKey=envVal")
	defer func() {
		os.Unsetenv(EnvOTLPEndpoint)
		os.Unsetenv(EnvOTLPInsecure)
		os.Unsetenv(EnvOTLPHeaders)
	}()

	cfg := newTestConfig()
	resolved := resolveOTLPConfig(cfg)

	assert.Equal(t, "env-collector:4317", resolved.Endpoint)
	assert.True(t, resolved.Insecure)
	assert.Equal(t, map[string]string{"envKey": "envVal"}, resolved.Headers)
}

func TestResolveOTLPConfig_DefaultToConfigValue(t *testing.T) {
	os.Unsetenv(EnvOTLPEndpoint)
	os.Unsetenv(EnvOTLPInsecure)
	os.Unsetenv(EnvOTLPHeaders)

	cfg := newTestConfig()
	resolved := resolveOTLPConfig(cfg)

	assert.Equal(t, "localhost:4317", resolved.Endpoint)
	assert.True(t, resolved.Insecure)
	assert.Equal(t, map[string]string{"x-api-key": "test-key"}, resolved.Headers)
}

func TestParseHeaderString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name:     "Single Key-Value Pair",
			input:    "key1=value1",
			expected: map[string]string{"key1": "value1"},
		},
		{
			name:     "Multiple Key-Value Pairs with Whitespace",
			input:    " key1 = value1 , key2=value2 ",
			expected: map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name:     "Invalid or Malformed Strings",
			input:    "invalid_pair,key3=value3",
			expected: map[string]string{"key3": "value3"},
		},
		{
			name:     "Empty String",
			input:    "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHeaderString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		input    string
		expected log.Severity
	}{
		{"trace", log.SeverityTrace},
		{"debug", log.SeverityDebug},
		{"DEBUG", log.SeverityDebug},
		{"info", log.SeverityInfo},
		{"warn", log.SeverityWarn},
		{"error", log.SeverityError},
		{"unknown", log.SeverityInfo}, // Default fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			severity := parseSeverity(tt.input)
			assert.Equal(t, tt.expected, severity)
		})
	}
}
