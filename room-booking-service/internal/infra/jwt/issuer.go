package jwt

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

type claims struct {
	UserID uuid.UUID       `json:"user_id"`
	Role   entity.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type issuer struct {
	secret []byte
	ttl    time.Duration
}

func NewIssuer(secret string, ttl time.Duration) (*issuer, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("jwt issuer - new: secret is required")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("jwt issuer - new: ttl must be positive")
	}

	return &issuer{
		secret: []byte(secret),
		ttl:    ttl,
	}, nil
}

func (issuer *issuer) Generate(userID uuid.UUID, role entity.UserRole) (string, error) {
	now := time.Now().UTC()

	claims := claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(issuer.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(issuer.secret)
	if err != nil {
		return "", fmt.Errorf("jwt issuer - generate: %w", err)
	}

	return signedToken, nil
}
