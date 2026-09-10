package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
	"go.uber.org/zap"
)

func Recovery(fallbackLogger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger := requestctx.LoggerOrDefault(c.Request.Context(), fallbackLogger).Named("http")
				logger.Error(
					"panic recovered",
					zap.String("method", c.Request.Method),
					zap.String("path", route(c)),
					zap.Any("panic", recovered),
					zap.ByteString("stack", debug.Stack()),
				)

				if !c.Writer.Written() {
					httperror.WriteError(
						c,
						http.StatusInternalServerError,
						httperror.CodeInternalError,
						"internal server error",
					)
				}

				c.Abort()
			}
		}()

		c.Next()
	}
}
