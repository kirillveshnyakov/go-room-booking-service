package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/config"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/handlers"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/middleware"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/conference"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/httpserver"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/jwt"
	loggerpkg "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/logger"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/password"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres/transactor"
	bookingDB "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/booking"
	roomDB "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/room"
	scheduleDB "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/schedule"
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

func Run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	logger, err := loggerpkg.New(loggerpkg.Config{
		Level:       cfg.Logger.Level,
		Environment: cfg.Logger.Environment,
		Service:     cfg.Logger.Service,
	})
	if err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	defer func() {
		if syncErr := logger.Sync(); syncErr != nil {
			logger.Warn("logger sync failed", zap.Error(syncErr))
		}
	}()

	poolCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(poolCtx, cfg.ConstructPostgresURL())
	if err != nil {
		return fmt.Errorf("initialize postgres pool: %w", err)
	}
	defer pool.Close()

	txManager := transactor.NewTransactor(pool)

	passwordHasher, err := password.NewHasher(cfg.Auth.PasswordHashCost)
	if err != nil {
		return fmt.Errorf("initialize password hasher: %w", err)
	}

	tokenIssuer, err := jwt.NewIssuer(cfg.Auth.JWTSecret, cfg.Auth.JWTTTL)
	if err != nil {
		return fmt.Errorf("initialize token issuer: %w", err)
	}

	linkGenerator, err := conference.NewLinkGenerator(cfg.Conference.BaseURL)
	if err != nil {
		return fmt.Errorf("initialize link generator: %w", err)
	}

	tokenVerifier, err := jwt.NewVerifier(cfg.Auth.JWTSecret)
	if err != nil {
		return fmt.Errorf("initialize token verifier: %w", err)
	}

	userRepository := userDB.NewUserRepository(pool)
	slotRepository := slotDB.NewSlotRepository(pool)
	roomRepository := roomDB.NewRoomRepository(pool)
	bookingRepository := bookingDB.NewBookingRepository(pool, txManager)
	scheduleRepository := scheduleDB.NewScheduleRepository(pool, txManager)

	authService := auth.NewAuthService(
		userRepository,
		passwordHasher,
		tokenIssuer,
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

	authHandler := handlers.NewAuthHandler(authService)
	roomHandler := handlers.NewRoomHandler(roomService)
	scheduleHandler := handlers.NewScheduleHandler(scheduleService)
	slotHandler := handlers.NewSlotHandler(slotService)
	bookingHandler := handlers.NewBookingHandler(bookingService)

	rateLimiter, err := middleware.NewUserRateLimiter(
		rate.Limit(cfg.HTTP.RateLimit),
		cfg.HTTP.RateLimitBurst,
	)
	if err != nil {
		return fmt.Errorf("initialize rate limiter: %w", err)
	}
	router := httpapi.NewRouter(
		authHandler,
		roomHandler,
		scheduleHandler,
		slotHandler,
		bookingHandler,
		middleware.Authentication(tokenVerifier, logger),
		middleware.RequestID(logger),
		middleware.Logging(logger),
		middleware.Recovery(logger),
		middleware.RateLimit(rateLimiter),
		middleware.ConcurrencyLimit(cfg.HTTP.ConcurrencyLimit),
	)

	server := httpserver.New(router, httpserver.Config{
		Host:              cfg.HTTP.Host,
		Port:              cfg.HTTP.Port,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	})

	if err = runHTTPServer(ctx, logger, cfg, server); err != nil {
		return fmt.Errorf("failed to run http server: %w", err)
	}

	return nil
}

func runHTTPServer(ctx context.Context, logger *zap.Logger, cfg *config.Config, server *httpserver.Server) error {
	serveErr := make(chan error, 1)

	go func() {
		logger.Info("http server starting", zap.String("address", server.Address()))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- fmt.Errorf("http server listen error: %w", err)
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.HTTP.HTTPShutdownTime)
	defer cancel()

	logger.Info("shutting down http server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Warn("http server shutdown error", zap.Error(err))
		if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			logger.Warn("http server forced close error", zap.Error(closeErr))
		}
		return fmt.Errorf("http server shutdown error: %w", err)
	}
	logger.Info("http server gracefully shutdown")

	return <-serveErr
}
