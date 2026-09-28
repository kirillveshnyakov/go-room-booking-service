package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/usecase/auth/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

type authMocks struct {
	users     *mocks.MockuserRepository
	sessions  *mocks.MocksessionRepository
	passwords *mocks.MockpasswordHasher
	access    *mocks.MockaccessTokenManager
	refresh   *mocks.MockrefreshTokenManager
}

func newAuthTestService(t *testing.T) (*authService, authMocks) {
	t.Helper()
	ctrl := gomock.NewController(t)
	m := authMocks{
		users:     mocks.NewMockuserRepository(ctrl),
		sessions:  mocks.NewMocksessionRepository(ctrl),
		passwords: mocks.NewMockpasswordHasher(ctrl),
		access:    mocks.NewMockaccessTokenManager(ctrl),
		refresh:   mocks.NewMockrefreshTokenManager(ctrl),
	}
	return NewAuthService(m.users, m.sessions, m.passwords, m.access, m.refresh, time.Hour, zap.NewNop()), m
}

func TestAuthService_Register(t *testing.T) {
	created := entity.User{ID: uuid.New(), Email: "user@example.com", Role: entity.UserRoleUser}
	technicalErr := errors.New("dependency unavailable")
	tests := []struct {
		name, email, password     string
		hashErr, repoErr, wantErr error
	}{
		{name: "success", email: "user@example.com", password: "password"},
		{name: "normalized email", email: " User@Example.COM\n", password: "password"},
		{name: "password spaces preserved", email: "user@example.com", password: " password "},
		{name: "72 byte password", email: "user@example.com", password: strings.Repeat("a", 72)},
		{name: "72 byte unicode password", email: "user@example.com", password: strings.Repeat("я", 36)},
		{name: "missing email", password: "password", wantErr: errs.ErrUserEmailRequired},
		{name: "whitespace email", email: " \n ", password: "password", wantErr: errs.ErrUserEmailRequired},
		{name: "missing password", email: "user@example.com", wantErr: errs.ErrPasswordRequired},
		{name: "73 byte password", email: "user@example.com", password: strings.Repeat("a", 73), wantErr: errs.ErrPasswordTooLong},
		{name: "unicode byte limit", email: "user@example.com", password: strings.Repeat("я", 37), wantErr: errs.ErrPasswordTooLong},
		{name: "hash error", email: "user@example.com", password: "password", hashErr: technicalErr, wantErr: technicalErr},
		{name: "duplicate email", email: "user@example.com", password: "password", repoErr: errs.ErrUserEmailAlreadyExists, wantErr: errs.ErrUserEmailAlreadyExists},
		{name: "wrapped duplicate email", email: "user@example.com", password: "password", repoErr: fmt.Errorf("create: %w", errs.ErrUserEmailAlreadyExists), wantErr: errs.ErrUserEmailAlreadyExists},
		{name: "repository error", email: "user@example.com", password: "password", repoErr: technicalErr, wantErr: technicalErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service, m := newAuthTestService(t)
			if tt.wantErr == nil || tt.hashErr != nil || tt.repoErr != nil {
				m.passwords.EXPECT().Hash(tt.password).Return("password-hash", tt.hashErr)
				if tt.hashErr == nil {
					m.users.EXPECT().Create(gomock.Any(), created.Email, entity.UserRoleUser, "password-hash").Return(created, tt.repoErr)
				}
			}
			got, err := service.Register(context.Background(), tt.email, tt.password)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, entity.User{}, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, created, got)
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	user := entity.User{ID: uuid.New(), Email: "user@example.com", Role: entity.UserRoleUser}
	hash := bytes.Repeat([]byte{1}, 32)
	technicalErr := errors.New("dependency unavailable")

	expectUserLookup := func(m authMocks, user entity.User, err error) {
		m.users.EXPECT().GetByEmail(gomock.Any(), "user@example.com").
			Return(entity.AuthUser{User: user, PasswordHash: "password-hash"}, err)
	}
	expectCredentials := func(m authMocks, user entity.User) {
		expectUserLookup(m, user, nil)
		m.passwords.EXPECT().Compare("password-hash", " password ").Return(nil)
	}
	expectTokens := func(t *testing.T, m authMocks, user entity.User, hash []byte, accessErr error) *uuid.UUID {
		t.Helper()
		var sessionID uuid.UUID
		var accessSessionID uuid.UUID
		t.Cleanup(func() {
			require.Equal(t, sessionID, accessSessionID)
		})
		expectCredentials(m, user)
		m.refresh.EXPECT().CreateRefreshToken(gomock.Any()).
			DoAndReturn(func(id uuid.UUID) (string, []byte, error) {
				require.NotEqual(t, uuid.Nil, id)
				sessionID = id
				return "refresh-token", hash, nil
			})
		m.access.EXPECT().CreateAccessToken(gomock.Any()).
			DoAndReturn(func(identity entity.Identity) (string, error) {
				require.Equal(t, user.ID, identity.UserID)
				require.Equal(t, user.Role, identity.Role)
				require.NotEqual(t, uuid.Nil, identity.SessionID)
				accessSessionID = identity.SessionID
				return "access-token", accessErr
			})
		return &sessionID
	}
	expectSessionCreate := func(t *testing.T, m authMocks, saved *entity.Session, err error) {
		t.Helper()
		before := time.Now().UTC()
		sessionID := expectTokens(t, m, user, hash, nil)
		t.Cleanup(func() {
			require.Equal(t, *sessionID, saved.ID)
		})
		m.sessions.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, session entity.Session) (entity.Session, error) {
				require.Equal(t, user.ID, session.UserID)
				require.Equal(t, hash, session.RefreshTokenHash)
				require.Nil(t, session.RevokedAt)
				require.Equal(t, time.UTC, session.ExpiresAt.Location())
				require.False(t, session.ExpiresAt.Before(before.Add(time.Hour)))
				require.False(t, session.ExpiresAt.After(time.Now().UTC().Add(time.Hour)))
				*saved = session
				return session, err
			})
	}

	tests := []struct {
		name       string
		setupMocks func(t *testing.T, m authMocks, saved *entity.Session)
		wantErr    error
	}{
		{
			name: "success",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectSessionCreate(t, m, saved, nil)
			},
		},
		{
			name: "user not found",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectUserLookup(m, user, errs.ErrUserNotFound)
			},
			wantErr: errs.ErrInvalidCredentials,
		},
		{
			name: "wrapped user not found",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectUserLookup(m, user, fmt.Errorf("lookup: %w", errs.ErrUserNotFound))
			},
			wantErr: errs.ErrInvalidCredentials,
		},
		{
			name: "user lookup error",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectUserLookup(m, user, technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name: "wrong password",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectUserLookup(m, user, nil)
				m.passwords.EXPECT().Compare("password-hash", " password ").Return(errs.ErrInvalidCredentials)
			},
			wantErr: errs.ErrInvalidCredentials,
		},
		{
			name: "wrapped wrong password",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectUserLookup(m, user, nil)
				m.passwords.EXPECT().Compare("password-hash", " password ").
					Return(fmt.Errorf("compare: %w", errs.ErrInvalidCredentials))
			},
			wantErr: errs.ErrInvalidCredentials,
		},
		{
			name: "password compare error",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectUserLookup(m, user, nil)
				m.passwords.EXPECT().Compare("password-hash", " password ").Return(technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name: "refresh generation error",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectCredentials(m, user)
				m.refresh.EXPECT().CreateRefreshToken(gomock.Any()).
					DoAndReturn(func(id uuid.UUID) (string, []byte, error) {
						require.NotEqual(t, uuid.Nil, id)
						return "", nil, technicalErr
					})
			},
			wantErr: technicalErr,
		},
		{
			name: "access generation error",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectTokens(t, m, user, hash, technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name: "missing user ID in session",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectTokens(t, m, entity.User{}, hash, nil)
			},
			wantErr: errs.ErrSessionUserIDRequired,
		},
		{
			name: "missing refresh hash",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectTokens(t, m, user, nil, nil)
			},
			wantErr: errs.ErrSessionRefreshHashRequired,
		},
		{
			name: "invalid refresh hash size",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectTokens(t, m, user, []byte{1}, nil)
			},
			wantErr: errs.ErrSessionRefreshHashInvalid,
		},
		{
			name: "session create error",
			setupMocks: func(t *testing.T, m authMocks, saved *entity.Session) {
				expectSessionCreate(t, m, saved, technicalErr)
			},
			wantErr: technicalErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service, m := newAuthTestService(t)
			var savedSession entity.Session
			tt.setupMocks(t, m, &savedSession)
			got, err := service.Login(context.Background(), " User@Example.COM ", " password ")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, port.AuthTokens{}, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, port.AuthTokens{AccessToken: "access-token", RefreshToken: "refresh-token", RefreshTokenExpiresAt: savedSession.ExpiresAt}, got)
		})
	}
}

func TestAuthService_Refresh(t *testing.T) {
	sessionID, userID := uuid.New(), uuid.New()
	expiresAt := time.Now().UTC().Add(time.Hour)
	revokedAt := time.Now().UTC().Add(-time.Minute)
	oldHash := bytes.Repeat([]byte{1}, 32)
	newHash := bytes.Repeat([]byte{2}, 32)
	session := entity.Session{ID: sessionID, UserID: userID, RefreshTokenHash: oldHash, ExpiresAt: expiresAt}
	technicalErr := errors.New("dependency unavailable")

	expectParse := func(m authMocks, err error) {
		m.refresh.EXPECT().ParseRefreshToken("old-refresh-token").Return(sessionID, oldHash, err)
	}
	expectSessionLookup := func(m authMocks, session entity.Session, err error) {
		expectParse(m, nil)
		m.sessions.EXPECT().GetByID(gomock.Any(), sessionID).Return(session, err)
	}
	expectUserLookup := func(m authMocks, role entity.UserRole, err error) {
		expectSessionLookup(m, session, nil)
		m.users.EXPECT().GetByID(gomock.Any(), userID).
			Return(entity.User{ID: userID, Role: role}, err)
	}
	expectTokens := func(m authMocks, role entity.UserRole, accessErr error) {
		expectUserLookup(m, role, nil)
		m.refresh.EXPECT().CreateRefreshToken(sessionID).Return("new-refresh-token", newHash, nil)
		m.access.EXPECT().CreateAccessToken(entity.Identity{UserID: userID, SessionID: sessionID, Role: role}).
			Return("new-access-token", accessErr)
	}
	expectRotation := func(m authMocks, role entity.UserRole, err error) {
		expectTokens(m, role, nil)
		m.sessions.EXPECT().RotateRefreshToken(gomock.Any(), sessionID, oldHash, newHash).Return(session, err)
	}

	tests := []struct {
		name       string
		setupMocks func(m authMocks)
		wantErr    error
	}{
		{
			name: "success user",
			setupMocks: func(m authMocks) {
				expectRotation(m, entity.UserRoleUser, nil)
			},
		},
		{
			name: "fresh role from repository",
			setupMocks: func(m authMocks) {
				expectRotation(m, entity.UserRoleAdmin, nil)
			},
		},
		{
			name: "invalid refresh token",
			setupMocks: func(m authMocks) {
				expectParse(m, errs.ErrInvalidRefreshToken)
			},
			wantErr: errs.ErrInvalidRefreshToken,
		},
		{
			name: "wrapped invalid refresh token",
			setupMocks: func(m authMocks) {
				expectParse(m, fmt.Errorf("parse: %w", errs.ErrInvalidRefreshToken))
			},
			wantErr: errs.ErrInvalidRefreshToken,
		},
		{
			name: "parse technical error",
			setupMocks: func(m authMocks) {
				expectParse(m, technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name: "session not found",
			setupMocks: func(m authMocks) {
				expectSessionLookup(m, entity.Session{}, errs.ErrSessionNotFound)
			},
			wantErr: errs.ErrInvalidRefreshToken,
		},
		{
			name: "wrapped session not found",
			setupMocks: func(m authMocks) {
				expectSessionLookup(m, entity.Session{}, fmt.Errorf("lookup: %w", errs.ErrSessionNotFound))
			},
			wantErr: errs.ErrInvalidRefreshToken,
		},
		{
			name: "session lookup error",
			setupMocks: func(m authMocks) {
				expectSessionLookup(m, entity.Session{}, technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name: "revoked session",
			setupMocks: func(m authMocks) {
				revoked := session
				revoked.RevokedAt = &revokedAt
				expectSessionLookup(m, revoked, nil)
			},
			wantErr: errs.ErrInvalidRefreshToken,
		},
		{
			name: "expired session",
			setupMocks: func(m authMocks) {
				expired := session
				expired.ExpiresAt = time.Now().UTC().Add(-time.Second)
				expectSessionLookup(m, expired, nil)
			},
			wantErr: errs.ErrInvalidRefreshToken,
		},
		{
			name: "zero expiry",
			setupMocks: func(m authMocks) {
				expired := session
				expired.ExpiresAt = time.Time{}
				expectSessionLookup(m, expired, nil)
			},
			wantErr: errs.ErrInvalidRefreshToken,
		},
		{
			name: "user not found",
			setupMocks: func(m authMocks) {
				expectUserLookup(m, entity.UserRoleUser, errs.ErrUserNotFound)
			},
			wantErr: errs.ErrInvalidRefreshToken,
		},
		{
			name: "wrapped user not found",
			setupMocks: func(m authMocks) {
				expectUserLookup(m, entity.UserRoleUser, fmt.Errorf("lookup: %w", errs.ErrUserNotFound))
			},
			wantErr: errs.ErrInvalidRefreshToken,
		},
		{
			name: "user lookup error",
			setupMocks: func(m authMocks) {
				expectUserLookup(m, entity.UserRoleUser, technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name: "refresh generation error",
			setupMocks: func(m authMocks) {
				expectUserLookup(m, entity.UserRoleUser, nil)
				m.refresh.EXPECT().CreateRefreshToken(sessionID).Return("", nil, technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name: "access generation error",
			setupMocks: func(m authMocks) {
				expectTokens(m, entity.UserRoleUser, technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name: "old token rejected during rotation",
			setupMocks: func(m authMocks) {
				expectRotation(m, entity.UserRoleUser, errs.ErrInvalidRefreshToken)
			},
			wantErr: errs.ErrInvalidRefreshToken,
		},
		{
			name: "wrapped rotation rejection",
			setupMocks: func(m authMocks) {
				expectRotation(m, entity.UserRoleUser, fmt.Errorf("rotate: %w", errs.ErrInvalidRefreshToken))
			},
			wantErr: errs.ErrInvalidRefreshToken,
		},
		{
			name: "rotation technical error",
			setupMocks: func(m authMocks) {
				expectRotation(m, entity.UserRoleUser, technicalErr)
			},
			wantErr: technicalErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service, m := newAuthTestService(t)
			tt.setupMocks(m)
			got, err := service.Refresh(context.Background(), "old-refresh-token")
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, port.AuthTokens{}, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, port.AuthTokens{AccessToken: "new-access-token", RefreshToken: "new-refresh-token", RefreshTokenExpiresAt: expiresAt}, got)
		})
	}
}

func TestAuthService_Logout(t *testing.T) {
	technicalErr := errors.New("database unavailable")
	tests := []struct {
		name             string
		id               uuid.UUID
		repoErr, wantErr error
	}{
		{name: "success", id: uuid.New()},
		{name: "missing session ID", wantErr: errs.ErrSessionIDRequired},
		{name: "repository error", id: uuid.New(), repoErr: technicalErr, wantErr: technicalErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service, m := newAuthTestService(t)
			before := time.Now().UTC()
			if tt.id != uuid.Nil {
				m.sessions.EXPECT().Revoke(gomock.Any(), tt.id, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, revokedAt time.Time) error {
					require.Equal(t, time.UTC, revokedAt.Location())
					require.False(t, revokedAt.Before(before))
					require.False(t, revokedAt.After(time.Now().UTC()))
					return tt.repoErr
				})
			}
			err := service.Logout(context.Background(), tt.id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAuthService_LogoutAll(t *testing.T) {
	technicalErr := errors.New("database unavailable")
	tests := []struct {
		name             string
		id               uuid.UUID
		repoErr, wantErr error
	}{
		{name: "success", id: uuid.New()},
		{name: "missing user ID", wantErr: errs.ErrIdentityUserIDRequired},
		{name: "repository error", id: uuid.New(), repoErr: technicalErr, wantErr: technicalErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service, m := newAuthTestService(t)
			before := time.Now().UTC()
			if tt.id != uuid.Nil {
				m.sessions.EXPECT().RevokeAllByUser(gomock.Any(), tt.id, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, revokedAt time.Time) error {
					require.Equal(t, time.UTC, revokedAt.Location())
					require.False(t, revokedAt.Before(before))
					require.False(t, revokedAt.After(time.Now().UTC()))
					return tt.repoErr
				})
			}
			err := service.LogoutAll(context.Background(), tt.id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAuthService_DummyLogin(t *testing.T) {
	technicalErr := errors.New("token manager unavailable")
	tests := []struct {
		name              string
		role              entity.UserRole
		userID            uuid.UUID
		tokenErr, wantErr error
	}{
		{name: "user", role: entity.UserRoleUser, userID: dummyUserID},
		{name: "admin", role: entity.UserRoleAdmin, userID: dummyAdminID},
		{name: "empty role", wantErr: errs.ErrUserRoleInvalid},
		{name: "invalid role", role: "owner", wantErr: errs.ErrUserRoleInvalid},
		{name: "generation error", role: entity.UserRoleUser, userID: dummyUserID, tokenErr: technicalErr, wantErr: technicalErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service, m := newAuthTestService(t)
			if tt.userID != uuid.Nil {
				m.access.EXPECT().CreateAccessToken(gomock.Any()).DoAndReturn(func(identity entity.Identity) (string, error) {
					require.Equal(t, tt.userID, identity.UserID)
					require.Equal(t, tt.role, identity.Role)
					require.NotEqual(t, uuid.Nil, identity.SessionID)
					return "access-token", tt.tokenErr
				})
			}
			got, err := service.DummyLogin(context.Background(), tt.role)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "access-token", got)
		})
	}
}
