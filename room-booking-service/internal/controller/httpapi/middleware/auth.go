package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/ctxvalues"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
	"go.uber.org/zap"
)

type tokenVerifier interface {
	Verify(string) (uuid.UUID, entity.UserRole, error)
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)

	if len(parts) != 2 ||
		!strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	return parts[1], true
}

func Authentication(verifier tokenVerifier, fallbackLogger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := bearerToken(c.Request.Header.Get("Authorization"))
		if !ok {
			unauthorized(c)
			return
		}

		userID, role, err := verifier.Verify(token)
		if err != nil {
			unauthorized(c)
			return
		}

		ctxvalues.SetUserID(c, userID)
		ctxvalues.SetRole(c, role)

		ctx := c.Request.Context()
		logger := requestctx.LoggerOrDefault(ctx, fallbackLogger).With(
			zap.String("user_id", userID.String()),
			zap.String("role", string(role)),
		)
		ctx = requestctx.WithLogger(ctx, logger)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	httperror.WriteError(
		c,
		http.StatusUnauthorized,
		httperror.CodeUnauthorized,
		"unauthorized",
	)

	c.Abort()
}
