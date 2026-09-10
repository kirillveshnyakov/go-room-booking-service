package ctxvalues

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

const (
	userIDKey    = "userID"
	roleKey      = "role"
	requestIDKey = "requestID"
)

func SetUserID(c *gin.Context, userID uuid.UUID) {
	c.Set(userIDKey, userID)
}

func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get(userIDKey)
	if !exists {
		return uuid.Nil, false
	}

	userID, ok := value.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, false
	}

	return userID, ok
}

func SetRole(c *gin.Context, role entity.UserRole) {
	c.Set(roleKey, role)
}

func GetRole(c *gin.Context) (entity.UserRole, bool) {
	value, exists := c.Get(roleKey)
	if !exists {
		return "", false
	}

	role, ok := value.(entity.UserRole)
	if !ok {
		return "", false
	}

	return role, ok
}

func SetRequestID(c *gin.Context, requestID string) {
	c.Set(requestIDKey, requestID)
}

func GetRequestID(c *gin.Context) (string, bool) {
	value, exists := c.Get(requestIDKey)
	if !exists {
		return "", false
	}

	requestID, ok := value.(string)
	if !ok || requestID == "" {
		return "", false
	}

	return requestID, ok
}
