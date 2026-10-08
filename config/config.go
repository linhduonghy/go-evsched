package config

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Service  Service    `mapstructure:"service"`
	TZ       string     `mapstructure:"timezone"`
	Server   Server     `mapstructure:"server"`
	Log      LogConfig  `mapstructure:"log"`
	Http     HttpConfig `mapstructure:"http"`
	DB       DBConfig   `mapstructure:"db"`
	OTel     OTelConfig `mapstructure:"otel"`
	RabbitMQ RabbitMQ   `mapstructure:"rabbitmq"`
	OutBox   Outbox     `mapstructure:"outbox"`
	Job      JobConfig  `mapstructure:"job"`
}

type Service struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
}

type Server struct {
	Port int `mapstructure:"port"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type HttpConfig struct {
	Timeout time.Duration `mapstructure:"timeout"`
}

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`

	MaxOpenConns int32 `mapstructure:"max_open_conns"`
	MinIdleConns int32 `mapstructure:"min_idle_conns"`
}

type OTelConfig struct {
	Endpoint string            `mapstructure:"endpoint"`
	Insecure bool              `mapstructure:"insecure"`
	Headers  map[string]string `mapstructure:"headers"`
	Trace    OtelTraceConfig   `mapstructure:"trace"`
	Log      OtelLogConfig     `mapstructure:"log"`
}

type OtelTraceConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type OtelLogConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type RabbitMQ struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	User string `mapstructure:"user"`
	Pass string `mapstructure:"pass"`
}

type Outbox struct {
	PollInterval time.Duration `mapstructure:"poll_interval"`
}
type JobConfig struct {
	JobStore JobStoreConfig `mapstructure:"store"`
}

type JobStoreConfig struct {
	MisfireThreshold time.Duration        `mapstructure:"misfire_threshold"`
	Publisher        map[string]Publisher `mapstructure:"publisher"`
}

type Publisher struct {
	Exchange   string `mapstructure:"exchange"`
	RoutingKey string `mapstructure:"routing_key"`
}

type ScheduledArg struct {
	Enabled  bool          `mapstructure:"enabled"`
	IsCron   bool          `mapstructure:"is_cron"`
	Cron     string        `mapstructure:"cron"`
	Interval time.Duration `mapstructure:"interval"`
}

func (c DBConfig) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Name,
		c.SSLMode,
	)
}

func (c RabbitMQ) URL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d", c.User, c.Pass, c.Host, c.Port)
}

func Load(defaultConfig string, overrideConfig string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	// Helper function to read file, expand environment variables, and parse with Viper
	loadAndExpandYAML := func(filePath string, merge bool) error {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read file %s: %w", filePath, err)
		}

		// Expand ${VAR:-default} or ${VAR} using environment variables with fallback support
		expandedData := expandEnvWithDefault(string(data))
		buf := bytes.NewBufferString(expandedData)

		if merge {
			return v.MergeConfig(buf)
		}
		return v.ReadConfig(buf)
	}

	// 1. Read base config.yaml
	if err := loadAndExpandYAML(defaultConfig, false); err != nil {
		return nil, fmt.Errorf("failed to read default config: %w", err)
	}

	// 2. Read override file if provided via -config flag
	if overrideConfig != "" {
		if err := loadAndExpandYAML(overrideConfig, true); err != nil {
			return nil, fmt.Errorf("failed to merge override config (%s): %w", overrideConfig, err)
		}
	}

	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	for _, key := range v.AllKeys() {
		_ = v.BindEnv(key)
	}

	var cfg Config

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Regex matches ${ENV_VAR:-fallback} or ${ENV_VAR}
var envPattern = regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)(?::-([^}]*))?}`)

// expandEnvWithDefault processes ${VAR:-default_value} or standard ${VAR} placeholders
func expandEnvWithDefault(s string) string {
	return envPattern.ReplaceAllStringFunc(s, func(match string) string {
		submatches := envPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		envVar := submatches[1]
		fallback := ""
		if len(submatches) > 2 {
			fallback = submatches[2]
		}

		if val, ok := os.LookupEnv(envVar); ok && val != "" {
			return val
		}

		return fallback
	})
}
