package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
)

func ConcurrencyLimit(limit int) gin.HandlerFunc {
	sem := make(chan struct{}, limit)

	return func(c *gin.Context) {
		select {
		case sem <- struct{}{}:
			defer func() {
				<-sem
			}()

			c.Next()

		default:
			httperror.WriteError(
				c,
				http.StatusServiceUnavailable,
				httperror.CodeServiceUnavailable,
				"service temporarily unavailable",
			)
			c.Abort()
		}
	}
}
