// Package config 负责应用配置的加载（环境变量优先）、默认值设定与启动前校验。
// 约定：环境变量优先于 .env 文件；非法配置会在启动阶段直接失败并给出清晰错误
package config

import (
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 表示应用运行时配置，来源于环境变量（若存在 .env 会被优先装载，但不会覆盖已存在的环境变量）。
// 建议的环境变量（与默认值）：
//   - APP_NAME=spike-server
//   - APP_ENV=dev|test|prod（默认 dev）
//   - APP_PORT（默认 8080）
//   - REQUEST_TIMEOUT_MS（默认 5000）
//   - LOG_LEVEL=debug|info|warn|error（默认 info）
//   - LOG_ENCODING=json|console（默认 json）
//   - CORS_ALLOWED_ORIGINS, CORS_ALLOWED_METHODS, CORS_ALLOWED_HEADERS（CSV）
type Config struct {
	App struct {
		Name            string
		Env             string
		Port            int
		RequestTimeout  time.Duration
		Version         string
		ShutdownTimeout time.Duration
	}

	Log struct {
		Level    string
		Encoding string
	}

	CORS struct {
		AllowedOrigins []string
		AllowedMethods []string
		AllowedHeaders []string
	}
}

// Load reads configuration from the environment (optionally loading a .env file if present),
// applies defaults, and validates the result.
func Load() (*Config, error) {
	// Load .env if present. Environment variables that are already set will NOT be overridden.
	_ = godotenv.Load()

	c := &Config{}

	c.App.Name = getEnv("APP_NAME", "Spike-server")
	c.App.Env = getEnv("APP_ENV", "dev")
	c.App.Port = getEnvAsInt("APP_PORT", 8080)
	c.App.RequestTimeout = getEnvAsDurationMs("REQUEST_TIMEOUT_MS", 5000)
	c.App.Version = getEnv("APP_VERSION", "0.1.0")
	c.App.ShutdownTimeout = getEnvAsDurationMs("SHUTDOWN_TIMEOUT_MS", 5000)

	c.Log.Level = strings.ToLower(getEnv("LOG_LEVEL", "info"))
	c.Log.Encoding = strings.ToLower(getEnv("LOG_ENCODING", "json"))

	c.CORS.AllowedOrigins = getEnvAsCSV("CORS_ALLOWED_ORIGINS", []string{"*"})
	c.CORS.AllowedMethods = getEnvAsCSV("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	c.CORS.AllowedHeaders = getEnvAsCSV("CORS_ALLOWED_HEADERS", []string{"Authorization", "Content-Type"})

	if err := validate(c); err != nil {
		return nil, err
	}
	return c, nil
}

func validate(c *Config) error {
	var errs []string

	switch c.App.Env {
	case "dev", "test", "prod":
		// ok
	default:
		errs = append(errs, fmt.Sprintf("APP_ENV must be one of dev|test|prod, got %q", c.App.Env))
	}

	if c.App.Port < 1 || c.App.Port > 65535 {
		errs = append(errs, fmt.Sprintf("APP_PORT must be in range 1..65535, got %d", c.App.Port))
	}

	if c.App.RequestTimeout <= 0 {
		errs = append(errs, fmt.Sprintf("REQUEST_TIMEOUT_MS must be > 0, got %s", c.App.RequestTimeout))
	}

	switch c.Log.Level {
	case "debug", "info", "warn", "error":
		// ok
	default:
		errs = append(errs, fmt.Sprintf("LOG_LEVEL must be one of debug|info|warn|error, got %q", c.Log.Level))
	}

	switch c.Log.Encoding {
	case "json", "console":
		// ok
	default:
		errs = append(errs, fmt.Sprintf("LOG_ENCODING must be one of json|console, got %q", c.Log.Encoding))
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return def
}

func getEnvAsInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return i
		}
	}
	return def
}

func getEnvAsDurationMs(key string, defMs int) time.Duration {
	ms := getEnvAsInt(key, defMs)
	return time.Duration(ms) * time.Millisecond
}

func getEnvAsCSV(key string, def []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return def
	}
	return out
}
