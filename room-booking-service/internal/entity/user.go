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

func (u *User) Normalize() {
	u.Email = strings.ToLower(normalizeText(u.Email))
	u.Role = u.Role.Normalize()
	u.CreatedAt = normalizeTime(u.CreatedAt)
}

func (u *User) Validate() error {
	u.Normalize()

	if !u.Role.IsValid() {
		return errs.ErrUserRoleInvalid
	}
	if u.Email == "" {
		return errs.ErrUserEmailRequired
	}
	return nil
}

type AuthUser struct {
	User         User
	PasswordHash string
}

func (u *AuthUser) Normalize() {
	u.User.Normalize()
}

func (u *AuthUser) Validate() error {
	return u.User.Validate()
}
