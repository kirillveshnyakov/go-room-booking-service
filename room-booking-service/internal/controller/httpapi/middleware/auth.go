package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/ctxvalues"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
	"go.uber.org/zap"
)

type tokenVerifier interface {
	VerifyAccessToken(string) (entity.Identity, error)
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

		identity, err := verifier.VerifyAccessToken(token)
		if err != nil {
			unauthorized(c)
			return
		}

		ctxvalues.SetUserID(c, identity.UserID)
		ctxvalues.SetSessionID(c, identity.SessionID)
		ctxvalues.SetRole(c, identity.Role)

		ctx := c.Request.Context()
		logger := requestctx.LoggerOrDefault(ctx, fallbackLogger).With(
			zap.String("user_id", identity.UserID.String()),
			zap.String("session_id", identity.SessionID.String()),
			zap.String("role", string(identity.Role)),
		)
		ctx = requestctx.WithIdentity(ctx, identity)
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
