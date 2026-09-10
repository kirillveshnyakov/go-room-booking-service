package requestctx

import (
	"context"

	"go.uber.org/zap"
)

type contextKey uint8

const (
	requestIDKey contextKey = iota
	loggerKey
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey).(string)
	if !ok || requestID == "" {
		return "", false
	}

	return requestID, true
}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func Logger(ctx context.Context) (*zap.Logger, bool) {
	logger, ok := ctx.Value(loggerKey).(*zap.Logger)
	if !ok || logger == nil {
		return nil, false
	}

	return logger, true
}

func LoggerOrDefault(ctx context.Context, fallback *zap.Logger) *zap.Logger {
	logger, ok := Logger(ctx)
	if !ok {
		return fallback
	}

	return logger
}
