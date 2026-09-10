package jwt

import (
	"fmt"
	"strings"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

type verifier struct {
	secret []byte
}

func NewVerifier(secret string) (*verifier, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("jwt verifier - new: secret is required")
	}

	return &verifier{secret: []byte(secret)}, nil
}

func (verifier *verifier) Verify(tokenString string) (uuid.UUID, entity.UserRole, error) {
	token, err := jwtlib.ParseWithClaims(
		tokenString,
		&claims{},
		func(token *jwtlib.Token) (any, error) {
			method, ok := token.Method.(*jwtlib.SigningMethodHMAC)
			if !ok || method.Alg() != jwtlib.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method %q", token.Header["alg"])
			}

			return verifier.secret, nil
		},
	)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("jwt verifier - verify: %w", err)
	}
	if !token.Valid {
		return uuid.Nil, "", fmt.Errorf("jwt verifier - verify: token is invalid")
	}

	parsedClaims, ok := token.Claims.(*claims)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("jwt verifier - verify: unexpected claims type")
	}
	if parsedClaims.UserID == uuid.Nil {
		return uuid.Nil, "", fmt.Errorf("jwt verifier - verify: user ID is required")
	}
	if !parsedClaims.Role.IsValid() {
		return uuid.Nil, "", fmt.Errorf("jwt verifier - verify: invalid user role")
	}

	return parsedClaims.UserID, parsedClaims.Role, nil
}
