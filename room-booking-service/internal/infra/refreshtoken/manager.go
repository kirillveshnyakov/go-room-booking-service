package refreshtoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
)

const refreshTokenSecretSize = 32

type tokenManager struct{}

func NewTokenManager() *tokenManager {
	return &tokenManager{}
}

func (manager *tokenManager) CreateRefreshToken(sessionID uuid.UUID) (string, []byte, error) {
	if sessionID == uuid.Nil {
		return "", nil, errs.ErrIdentitySessionIDRequired
	}

	secret := make([]byte, refreshTokenSecretSize)

	if _, err := rand.Read(secret); err != nil {
		return "", nil, fmt.Errorf("refresh token manager - create: generate secret: %w", err)
	}

	hash := sha256.Sum256(secret)

	encoded := base64.RawURLEncoding.EncodeToString(secret)

	token := sessionID.String() + "." + encoded

	return token, hash[:], nil
}

func (manager *tokenManager) ParseRefreshToken(token string) (uuid.UUID, []byte, error) {
	parts := strings.SplitN(token, ".", 2)

	if len(parts) != 2 {
		return uuid.Nil, nil, errs.ErrInvalidRefreshToken
	}

	sessionID, err := uuid.Parse(parts[0])
	if err != nil || sessionID == uuid.Nil {
		return uuid.Nil, nil, errs.ErrInvalidRefreshToken
	}

	secret, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(secret) != refreshTokenSecretSize {
		return uuid.Nil, nil, errs.ErrInvalidRefreshToken
	}

	hash := sha256.Sum256(secret)

	return sessionID, hash[:], nil
}
