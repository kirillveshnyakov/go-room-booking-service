package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/ctxvalues"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
	"go.uber.org/zap"
)

const requestIDHeader = "X-Request-ID"

func RequestID(rootLogger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.NewString()
		requestLogger := rootLogger.With(zap.String("request_id", requestID))

		ctxvalues.SetRequestID(c, requestID)

		ctx := c.Request.Context()
		ctx = requestctx.WithRequestID(ctx, requestID)
		ctx = requestctx.WithLogger(ctx, requestLogger)
		c.Request = c.Request.WithContext(ctx)

		c.Header(requestIDHeader, requestID)

		c.Next()
	}
}
