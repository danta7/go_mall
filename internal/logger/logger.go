package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

// New 根据 env/level/encoding 构建 *zap.Logger。
// - env: dev|test|prod（dev 使用 DevelopmentConfig，prod 使用 ProductionConfig）
// - level: debug|info|warn|error
// - encoding: json|console（生产建议 json）
func New(env, level, encoding, serviceName, version string) (*zap.Logger, error) {
	var cfg zap.Config
	if env == "prod" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	// Level
	switch level {
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		cfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Encoding
	if encoding == "console" {
		cfg.Encoding = "console"
	} else {
		cfg.Encoding = "json"
	}

	// Output to stdout/stderr
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	// Production JSON should use ISO8601 time and lowercase level
	cfg.EncoderConfig = zap.NewProductionEncoderConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.MessageKey = "msg"
	cfg.EncoderConfig.CallerKey = "caller"

	lg, err := cfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	// common fields
	lg = lg.With(
		zap.String("service", serviceName),
		zap.String("version", version),
		zap.String("env", env),
		zap.String("pid", fmt.Sprintf("%d", os.Getpid())),
	)

	return lg, nil
}
