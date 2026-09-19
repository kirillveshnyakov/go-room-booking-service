package ctxvalues

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

const (
	userIDKey    = "userID"
	sessionIDKey = "sessionID"
	roleKey      = "role"
	requestIDKey = "requestID"
)

func SetUserID(c *gin.Context, userID uuid.UUID) {
	c.Set(userIDKey, userID)
}

func SetSessionID(c *gin.Context, sessionID uuid.UUID) {
	c.Set(sessionIDKey, sessionID)
}

func GetSessionID(c *gin.Context) (uuid.UUID, bool) {
	value, _ := c.Get(sessionIDKey)
	sessionID, ok := value.(uuid.UUID)
	return sessionID, ok && sessionID != uuid.Nil
}

func GetUserID(c *gin.Context) (uuid.UUID, bool) {
	value, _ := c.Get(userIDKey)
	userID, ok := value.(uuid.UUID)
	return userID, ok && userID != uuid.Nil
}

func SetRole(c *gin.Context, role entity.UserRole) {
	c.Set(roleKey, role)
}

func GetRole(c *gin.Context) (entity.UserRole, bool) {
	value, _ := c.Get(roleKey)
	role, ok := value.(entity.UserRole)
	return role, ok
}

func SetRequestID(c *gin.Context, requestID string) {
	c.Set(requestIDKey, requestID)
}

func GetRequestID(c *gin.Context) (string, bool) {
	value, _ := c.Get(requestIDKey)
	requestID, ok := value.(string)
	return requestID, ok && requestID != ""
}
