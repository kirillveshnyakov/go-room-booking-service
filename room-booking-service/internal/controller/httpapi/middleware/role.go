package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/ctxvalues"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

func RequireRole(requiredRole entity.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := ctxvalues.GetRole(c)
		if !ok {
			httperror.WriteError(
				c,
				http.StatusUnauthorized,
				httperror.CodeUnauthorized,
				"unauthorized",
			)
			c.Abort()
			return
		}

		if role != requiredRole {
			httperror.WriteError(
				c,
				http.StatusForbidden,
				httperror.CodeForbidden,
				"forbidden",
			)
			c.Abort()
			return
		}

		c.Next()
	}
}
