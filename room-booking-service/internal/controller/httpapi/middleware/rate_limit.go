package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/ctxvalues"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/httperror"
	"golang.org/x/time/rate"
)

type limiter interface {
	Allow(userID uuid.UUID) bool
}

const (
	limiterEntryTTL        = 10 * time.Minute
	limiterCleanupInterval = time.Minute
)

type userRateLimiter struct {
	limit rate.Limit
	burst int

	mu          sync.Mutex
	entries     map[uuid.UUID]limiterEntry
	nextCleanup time.Time
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type globalRateLimiter struct {
	limiter *rate.Limiter
}

func NewUserRateLimiter(limit rate.Limit, burst int) (*userRateLimiter, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("rate limit must be positive")
	}
	if burst <= 0 {
		return nil, fmt.Errorf("rate limit burst must be positive")
	}

	return &userRateLimiter{
		limit:   limit,
		burst:   burst,
		entries: make(map[uuid.UUID]limiterEntry),
	}, nil
}

func (limiter *userRateLimiter) Allow(userID uuid.UUID) bool {
	now := time.Now()

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if now.After(limiter.nextCleanup) {
		for id, entry := range limiter.entries {
			if now.Sub(entry.lastSeen) >= limiterEntryTTL {
				delete(limiter.entries, id)
			}
		}
		limiter.nextCleanup = now.Add(limiterCleanupInterval)
	}

	entry, exists := limiter.entries[userID]
	if !exists {
		entry.limiter = rate.NewLimiter(limiter.limit, limiter.burst)
	}

	entry.lastSeen = now
	limiter.entries[userID] = entry

	return entry.limiter.Allow()
}

func NewGlobalRateLimiter(limit rate.Limit, burst int) (*globalRateLimiter, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("rate limit must be positive")
	}
	if burst <= 0 {
		return nil, fmt.Errorf("rate limit burst must be positive")
	}

	return &globalRateLimiter{
		limiter: rate.NewLimiter(limit, burst),
	}, nil
}

func (limiter *globalRateLimiter) Allow() bool {
	return limiter.limiter.Allow()
}

func RateLimit(l limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := ctxvalues.GetUserID(c)
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

		if !l.Allow(userID) {
			httperror.WriteError(
				c,
				http.StatusTooManyRequests,
				httperror.CodeTooManyRequests,
				"too many requests",
			)
			c.Abort()
			return
		}

		c.Next()
	}
}

func GlobalRateLimit(l *globalRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow() {
			httperror.WriteError(
				c,
				http.StatusTooManyRequests,
				httperror.CodeTooManyRequests,
				"too many requests",
			)
			c.Abort()
			return
		}

		c.Next()
	}
}
