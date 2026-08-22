package password

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
)

type hasher struct {
	cost int
}

func NewHasher(cost int) (*hasher, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, fmt.Errorf(
			"password hasher - new: %w",
			bcrypt.InvalidCostError(cost),
		)
	}

	return &hasher{cost: cost}, nil
}

func (hasher *hasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), hasher.cost)
	if err != nil {
		return "", fmt.Errorf("password hasher - hash: %w", err)
	}

	return string(hash), nil
}

func (hasher *hasher) Compare(hash string, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		return nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return errs.ErrInvalidCredentials
	}

	return fmt.Errorf("password hasher - compare: %w", err)
}
