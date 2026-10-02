package app

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/config"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/handlers"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/middleware"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/conference"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/jwt"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/password"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres/transactor"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/refreshtoken"
	bookingDB "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/booking"
	roomDB "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/room"
	scheduleDB "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/schedule"
	sessionDB "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/session"
	slotDB "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/slot"
	userDB "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/user"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/usecase/auth"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/usecase/booking"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/usecase/room"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/usecase/schedule"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/usecase/slot"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func NewApplication(
	cfg *config.Config,
	pool *pgxpool.Pool,
	logger *zap.Logger,
) (http.Handler, error) {
	txManager := transactor.NewTransactor(pool)

	passwordHasher, err := password.NewHasher(cfg.Auth.PasswordHashCost)
	if err != nil {
		return nil, fmt.Errorf("initialize password hasher: %w", err)
	}

	accessTokenManager, err := jwt.NewTokenManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.AccessTokenIssuer,
		cfg.Auth.AccessTokenAudience,
		cfg.Auth.AccessTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize access token manager: %w", err)
	}
	refreshTokenManager := refreshtoken.NewTokenManager()

	linkGenerator, err := conference.NewLinkGenerator(cfg.Conference.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("initialize link generator: %w", err)
	}

	userRepository := userDB.NewUserRepository(pool)
	slotRepository := slotDB.NewSlotRepository(pool)
	roomRepository := roomDB.NewRoomRepository(pool)
	bookingRepository := bookingDB.NewBookingRepository(pool, txManager)
	scheduleRepository := scheduleDB.NewScheduleRepository(pool, txManager)
	sessionRepository := sessionDB.NewSessionRepository(pool)

	authService := auth.NewAuthService(
		userRepository,
		sessionRepository,
		passwordHasher,
		accessTokenManager,
		refreshTokenManager,
		cfg.Auth.SessionTTL,
		logger,
	)
	roomService := room.NewRoomService(
		roomRepository,
		logger,
	)
	slotService := slot.NewSlotService(
		slotRepository,
		scheduleRepository,
		roomRepository,
		logger,
	)
	scheduleService := schedule.NewScheduleService(
		scheduleRepository,
		logger,
	)
	bookingService := booking.NewBookingService(
		bookingRepository,
		linkGenerator,
		logger,
	)

	authHandler := handlers.NewAuthHandler(
		authService,
		handlers.RefreshCookieConfig{
			Name:     cfg.Auth.RefreshCookieName,
			Path:     cfg.Auth.RefreshCookiePath,
			Secure:   cfg.Auth.RefreshCookieSecure,
			SameSite: parseSameSite(cfg.Auth.RefreshCookieSameSite),
			TTL:      cfg.Auth.SessionTTL,
		},
	)
	roomHandler := handlers.NewRoomHandler(roomService)
	scheduleHandler := handlers.NewScheduleHandler(scheduleService)
	slotHandler := handlers.NewSlotHandler(slotService)
	bookingHandler := handlers.NewBookingHandler(bookingService)

	rateLimiter, err := middleware.NewUserRateLimiter(
		rate.Limit(cfg.HTTP.RateLimit),
		cfg.HTTP.RateLimitBurst,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize rate limiter: %w", err)
	}

	authRateLimiter, err := middleware.NewGlobalRateLimiter(
		rate.Limit(cfg.Auth.AuthRateLimit),
		cfg.Auth.AuthRateLimitBurst,
	)
	if err != nil {
		return nil, fmt.Errorf("initialize auth rate limiter: %w", err)
	}

	router := httpapi.NewRouter(
		authHandler,
		roomHandler,
		scheduleHandler,
		slotHandler,
		bookingHandler,
		middleware.Authentication(accessTokenManager, logger),
		middleware.RequestID(logger),
		middleware.Logging(logger),
		middleware.Recovery(logger),
		middleware.RateLimit(rateLimiter),
		middleware.GlobalRateLimit(authRateLimiter),
		middleware.ConcurrencyLimit(cfg.HTTP.ConcurrencyLimit),
		cfg,
	)

	return router, nil
}

func parseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
