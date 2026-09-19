package middleware

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
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

type ipRateLimiter struct {
	limit rate.Limit
	burst int

	mu          sync.Mutex
	entries     map[netip.Addr]limiterEntry
	nextCleanup time.Time
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

func NewIPRateLimiter(limit rate.Limit, burst int) (*ipRateLimiter, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("rate limit must be positive")
	}
	if burst <= 0 {
		return nil, fmt.Errorf("rate limit burst must be positive")
	}

	return &ipRateLimiter{
		limit:   limit,
		burst:   burst,
		entries: make(map[netip.Addr]limiterEntry),
	}, nil
}

func (limiter *ipRateLimiter) Allow(clientIP netip.Addr) bool {
	now := time.Now()

	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if now.After(limiter.nextCleanup) {
		for ip, entry := range limiter.entries {
			if now.Sub(entry.lastSeen) >= limiterEntryTTL {
				delete(limiter.entries, ip)
			}
		}
		limiter.nextCleanup = now.Add(limiterCleanupInterval)
	}

	entry, exists := limiter.entries[clientIP]
	if !exists {
		entry.limiter = rate.NewLimiter(limiter.limit, limiter.burst)
	}

	entry.lastSeen = now
	limiter.entries[clientIP] = entry

	return entry.limiter.Allow()
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

func IPRateLimit(l *ipRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err != nil {
			httperror.WriteError(
				c,
				http.StatusBadRequest,
				httperror.CodeInvalidRequest,
				"invalid client IP address",
			)
			c.Abort()
			return
		}

		clientIP, err := netip.ParseAddr(host)
		if err != nil {
			httperror.WriteError(
				c,
				http.StatusBadRequest,
				httperror.CodeInvalidRequest,
				"invalid client IP address",
			)
			c.Abort()
			return
		}

		if !l.Allow(clientIP) {
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
