package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/config"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/handlers"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/middleware"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

func NewRouter(
	authHandler *handlers.AuthHandler,
	roomHandler *handlers.RoomHandler,
	scheduleHandler *handlers.ScheduleHandler,
	slotHandler *handlers.SlotHandler,
	bookingHandler *handlers.BookingHandler,

	authMiddleware gin.HandlerFunc,
	requestIDMiddleware gin.HandlerFunc,
	loggingMiddleware gin.HandlerFunc,
	recoveryMiddleware gin.HandlerFunc,
	rateLimiterMiddleware gin.HandlerFunc,
	authRateLimiterMiddleware gin.HandlerFunc,
	concurrencyLimiterMiddleware gin.HandlerFunc,

	cfg *config.Config,
) *gin.Engine {
	router := gin.New()

	router.Use(
		requestIDMiddleware,
		loggingMiddleware,
		recoveryMiddleware,
	)
	router.GET("/_info", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Public
	router.POST("/register", authRateLimiterMiddleware, authHandler.Register)
	router.POST("/login", authRateLimiterMiddleware, authHandler.Login)
	router.POST("/refresh", authRateLimiterMiddleware, authHandler.Refresh)

	if cfg.Auth.EnableDummyLogin {
		router.POST("/dummyLogin", authRateLimiterMiddleware, authHandler.DummyLogin)
	}

	// Protected
	protected := router.Group("/")
	protected.Use(authMiddleware)

	protected.POST("/logout", authHandler.Logout)
	protected.POST("/logout-all", authHandler.LogoutAll)
	protected.GET("/rooms/list", roomHandler.List)
	protected.POST("/bookings/:bookingId/cancel", bookingHandler.Cancel)

	protected.GET(
		"/rooms/:roomId/slots/list",
		rateLimiterMiddleware,
		concurrencyLimiterMiddleware,
		slotHandler.List,
	)

	// Admin
	admin := protected.Group("/")
	admin.Use(middleware.RequireRole(entity.UserRoleAdmin))

	admin.POST("/rooms/create", roomHandler.Create)
	admin.POST("/rooms/:roomId/schedule/create", scheduleHandler.Create)
	admin.GET("/bookings/list", bookingHandler.List)

	// User
	user := protected.Group("/")
	user.Use(middleware.RequireRole(entity.UserRoleUser))

	user.POST("/bookings/create", bookingHandler.Create)
	user.GET("/bookings/my", bookingHandler.My)

	return router
}
