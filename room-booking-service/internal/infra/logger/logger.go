package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Level       string
	Environment string
	Service     string
}

func New(cfg Config) (*zap.Logger, error) {
	var zapCfg zap.Config

	switch cfg.Environment {
	case "development":
		zapCfg = zap.NewDevelopmentConfig()
	case "production":
		zapCfg = zap.NewProductionConfig()
	default:
		return nil, fmt.Errorf("unknown environment: %s", cfg.Environment)
	}

	level, err := zap.ParseAtomicLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}

	zapCfg.Level = level
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	zapCfg.OutputPaths = []string{"stdout"}
	zapCfg.ErrorOutputPaths = []string{"stderr"}

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("build zap logger: %w", err)
	}

	logger = logger.With(
		zap.String("service", cfg.Service),
		zap.String("environment", cfg.Environment),
	)

	return logger, nil
}
