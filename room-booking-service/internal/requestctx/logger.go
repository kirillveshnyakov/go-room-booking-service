package requestctx

import (
	"context"

	"go.uber.org/zap"
)

type loggerKey struct{}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func Logger(ctx context.Context) (*zap.Logger, bool) {
	logger, ok := ctx.Value(loggerKey{}).(*zap.Logger)
	return logger, ok && logger != nil
}

func LoggerOrDefault(ctx context.Context, fallback *zap.Logger) *zap.Logger {
	logger, ok := Logger(ctx)
	if !ok {
		return fallback
	}

	return logger
}
