package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
	"go.uber.org/zap"
)

func Logging(fallbackLogger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		logger := requestctx.LoggerOrDefault(c.Request.Context(), fallbackLogger).Named("http")
		logger.Info("request completed",
			zap.String("method", c.Request.Method),
			zap.String("path", route(c)),
			zap.Int("status", c.Writer.Status()),
			zap.Int("response_size", c.Writer.Size()),
			zap.Duration("duration", time.Since(start)),
		)
	}
}

func route(c *gin.Context) string {
	if route := c.FullPath(); route != "" {
		return route
	}

	return c.Request.URL.Path
}
