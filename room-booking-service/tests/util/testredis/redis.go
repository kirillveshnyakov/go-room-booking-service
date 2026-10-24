package testredis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

type Redis struct {
	Client    *redis.Client
	container *rediscontainer.RedisContainer
}

func Start(ctx context.Context) (result *Redis, err error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	server := &Redis{}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("start test Redis: Docker initialization failed: %v", recovered)
		}
		if err != nil {
			err = errors.Join(err, server.Close())
			result = nil
		}
	}()

	server.container, err = rediscontainer.Run(ctx, "redis:8-alpine")
	if err != nil {
		return nil, fmt.Errorf("start test Redis container: %w", err)
	}
	uri, err := server.container.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("get test Redis address: %w", err)
	}
	options, err := redis.ParseURL(uri)
	if err != nil {
		return nil, fmt.Errorf("parse test Redis address: %w", err)
	}
	options.DialTimeout = 3 * time.Second
	options.ReadTimeout = 3 * time.Second
	options.WriteTimeout = 3 * time.Second
	options.ContextTimeoutEnabled = true
	options.MaxRetries = -1
	server.Client = redis.NewClient(options)
	pingCtx, pingCancel := context.WithTimeout(ctx, 5*time.Second)
	defer pingCancel()
	if err = server.Client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("ping test Redis: %w", err)
	}
	return server, nil
}

func (server *Redis) Close() error {
	var cleanupErr error
	if server.Client != nil {
		if err := server.Client.Close(); err != nil {
			cleanupErr = fmt.Errorf("close test Redis client: %w", err)
		}
	}
	if server.container != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.container.Terminate(ctx); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("terminate test Redis container: %w", err))
		}
	}
	return cleanupErr
}
