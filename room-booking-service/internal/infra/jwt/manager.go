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
	SessionID string          `json:"sid"`
	Role      entity.UserRole `json:"role"`
	jwt.RegisteredClaims
}

type tokenManager struct {
	secret   []byte
	ttl      time.Duration
	issuer   string
	audience string
}

func NewTokenManager(
	secret string,
	issuer string,
	audience string,
	ttl time.Duration,
) (*tokenManager, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("jwt token manager - new: secret is required")
	}
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		return nil, fmt.Errorf("jwt token manager - new: issuer is required")
	}
	audience = strings.TrimSpace(audience)
	if audience == "" {
		return nil, fmt.Errorf("jwt token manager - new: audience is required")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("jwt token manager - new: ttl must be positive")
	}

	return &tokenManager{
		secret:   []byte(secret),
		ttl:      ttl,
		issuer:   issuer,
		audience: audience,
	}, nil
}

func (manager *tokenManager) GenerateToken(identity entity.Identity) (string, error) {
	if err := identity.Validate(); err != nil {
		return "", fmt.Errorf("jwt token manager - generate token: validate identity: %w", err)
	}

	now := time.Now().UTC()

	claims := claims{
		SessionID: identity.SessionID.String(),
		Role:      identity.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   identity.UserID.String(),
			Issuer:    manager.issuer,
			Audience:  jwt.ClaimStrings{manager.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(manager.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(manager.secret)
	if err != nil {
		return "", fmt.Errorf("jwt token manager - generate token: %w", err)
	}

	return signedToken, nil
}

func (manager *tokenManager) VerifyToken(tokenString string) (entity.Identity, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claims{},
		func(token *jwt.Token) (any, error) {
			return manager.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithIssuer(manager.issuer),
		jwt.WithAudience(manager.audience),
	)
	if err != nil {
		return entity.Identity{}, fmt.Errorf("jwt token manager - verify token: %w", err)
	}
	if !token.Valid {
		return entity.Identity{}, fmt.Errorf("jwt token manager - verify token: token is invalid")
	}

	parsedClaims, ok := token.Claims.(*claims)
	if !ok {
		return entity.Identity{}, fmt.Errorf("jwt token manager - verify token: unexpected claims type")
	}
	if parsedClaims.IssuedAt == nil {
		return entity.Identity{}, fmt.Errorf("jwt token manager - verify token: issued at is required")
	}

	userID, err := uuid.Parse(parsedClaims.Subject)
	if err != nil {
		return entity.Identity{}, fmt.Errorf("jwt token manager - verify token: parse subject: %w", err)
	}
	sessionID, err := uuid.Parse(parsedClaims.SessionID)
	if err != nil {
		return entity.Identity{}, fmt.Errorf("jwt token manager - verify token: parse session ID: %w", err)
	}

	identity := entity.Identity{
		UserID:    userID,
		SessionID: sessionID,
		Role:      parsedClaims.Role,
	}
	if err = identity.Validate(); err != nil {
		return entity.Identity{}, fmt.Errorf("jwt token manager - verify token: validate identity: %w", err)
	}

	return identity, nil
}
