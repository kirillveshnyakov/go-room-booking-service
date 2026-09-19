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

type Identity struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Role      UserRole
}

func (i *Identity) Normalize() {
	i.Role = i.Role.Normalize()
}

func (i *Identity) Validate() error {
	i.Normalize()

	if i.UserID == uuid.Nil {
		return errs.ErrIdentityUserIDRequired
	}
	if i.SessionID == uuid.Nil {
		return errs.ErrIdentitySessionIDRequired
	}
	if !i.Role.IsValid() {
		return errs.ErrUserRoleInvalid
	}

	return nil
}

type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash []byte

	ExpiresAt time.Time
	RevokedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

const sessionRefreshTokenHashSize = 32

func (s *Session) Normalize() {
	s.CreatedAt = normalizeTime(s.CreatedAt)
	s.UpdatedAt = normalizeTime(s.UpdatedAt)
	s.ExpiresAt = normalizeTime(s.ExpiresAt)
	if s.RevokedAt != nil {
		*s.RevokedAt = normalizeTime(*s.RevokedAt)
	}
}

func (s *Session) Validate() error {
	s.Normalize()

	if s.ID == uuid.Nil {
		return errs.ErrSessionIDRequired
	}
	if s.UserID == uuid.Nil {
		return errs.ErrSessionUserIDRequired
	}

	if len(s.RefreshTokenHash) == 0 {
		return errs.ErrSessionRefreshHashRequired
	}
	if len(s.RefreshTokenHash) != sessionRefreshTokenHashSize {
		return errs.ErrSessionRefreshHashInvalid
	}

	if s.ExpiresAt.IsZero() {
		return errs.ErrSessionExpiresAtRequired
	}

	return nil
}

func (s *Session) IsExpired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}

func (s *Session) IsRevoked() bool {
	return s.RevokedAt != nil
}
