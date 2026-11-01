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
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/httpserver"
	loggerpkg "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/logger"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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
		_ = logger.Sync()
	}()

	poolCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(poolCtx, cfg.ConstructPostgresURL())
	if err != nil {
		return fmt.Errorf("initialize postgres pool: %w", err)
	}
	defer pool.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr:                  cfg.Redis.Address(),
		Username:              cfg.Redis.Username,
		Password:              cfg.Redis.Password,
		DB:                    cfg.Redis.DB,
		DialTimeout:           cfg.Redis.DialTimeout,
		ReadTimeout:           cfg.Redis.ReadTimeout,
		WriteTimeout:          cfg.Redis.WriteTimeout,
		ContextTimeoutEnabled: true,
		MaxRetries:            -1,
		DialerRetries:         1,
	})
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Warn("redis client close failed", zap.Error(err))
		}
	}()

	redisPingCtx, cancelRedisPing := context.WithTimeout(ctx, cfg.Redis.DialTimeout)
	redisPingErr := redisClient.Ping(redisPingCtx).Err()
	cancelRedisPing()
	if redisPingErr != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("initialize redis client: %w", ctx.Err())
		}
		logger.Warn("redis unavailable at startup", zap.Error(redisPingErr))
	}

	handler, err := NewApplication(cfg, pool, redisClient, logger)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}

	server := httpserver.New(handler, httpserver.Config{
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
