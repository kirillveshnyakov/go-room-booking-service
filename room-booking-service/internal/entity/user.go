package entity

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
)

type User struct {
	ID        uuid.UUID
	Email     string
	Role      UserRole
	CreatedAt time.Time
}

func (u User) Validate() error {
	if !u.Role.IsValid() {
		return errs.ErrUserRoleInvalid
	}
	if strings.TrimSpace(u.Email) == "" {
		return errs.ErrUserEmailRequired
	}
	return nil
}

type AuthUser struct {
	User         User
	PasswordHash string
}

func (u AuthUser) Validate() error {
	return u.User.Validate()
}
