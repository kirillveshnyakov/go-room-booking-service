package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
	"go.uber.org/zap"
)

type (
	userRepository interface {
		Create(ctx context.Context, email string, role entity.UserRole, passwordHash string) (entity.User, error)
		GetByEmail(ctx context.Context, email string) (entity.AuthUser, error)
		GetByID(ctx context.Context, userID uuid.UUID) (entity.User, error)
	}

	sessionRepository interface {
		Create(ctx context.Context, session entity.Session) (entity.Session, error)
		GetByID(ctx context.Context, sessionID uuid.UUID) (entity.Session, error)
		RotateRefreshToken(
			ctx context.Context,
			sessionID uuid.UUID,
			oldRefreshTokenHash []byte,
			newRefreshTokenHash []byte,
		) (entity.Session, error)
		Revoke(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error
		RevokeAllByUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
	}

	passwordHasher interface {
		Hash(password string) (string, error)
		Compare(hash, password string) error
	}

	accessTokenManager interface {
		CreateAccessToken(identity entity.Identity) (string, error)
	}

	refreshTokenManager interface {
		CreateRefreshToken(sessionID uuid.UUID) (rawToken string, hash []byte, err error)
		ParseRefreshToken(rawToken string) (sessionID uuid.UUID, hash []byte, err error)
	}
)

type authService struct {
	userRepository      userRepository
	sessionRepository   sessionRepository
	passwordHasher      passwordHasher
	accessTokenManager  accessTokenManager
	refreshTokenManager refreshTokenManager
	sessionTTL          time.Duration
	logger              *zap.Logger
}

func NewAuthService(
	userRepository userRepository,
	sessionRepository sessionRepository,
	passwordHasher passwordHasher,
	accessTokenManager accessTokenManager,
	refreshTokenManager refreshTokenManager,
	sessionTTL time.Duration,
	logger *zap.Logger,
) *authService {
	return &authService{
		userRepository:      userRepository,
		sessionRepository:   sessionRepository,
		passwordHasher:      passwordHasher,
		accessTokenManager:  accessTokenManager,
		refreshTokenManager: refreshTokenManager,
		sessionTTL:          sessionTTL,
		logger:              logger,
	}
}

func (service *authService) Register(
	ctx context.Context,
	email string,
	password string,
) (entity.User, error) {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("auth_usecase")

	user := entity.User{
		Email: email,
		Role:  entity.UserRoleUser,
	}
	if err := user.Validate(); err != nil {
		return entity.User{}, fmt.Errorf("auth usecase - register: validation error: %w", err)
	}
	if err := validatePassword(password); err != nil {
		return entity.User{}, fmt.Errorf("auth usecase - register: validation error: %w", err)
	}

	passwordHash, err := service.passwordHasher.Hash(password)
	if err != nil {
		log.Error(
			"user registration failed",
			zap.Error(err),
		)

		return entity.User{}, fmt.Errorf("auth usecase - register: %w", err)
	}

	createdUser, createErr := service.userRepository.Create(ctx, user.Email, user.Role, passwordHash)
	if createErr != nil {
		if errors.Is(createErr, errs.ErrUserEmailAlreadyExists) {
			return entity.User{}, createErr
		}

		log.Error(
			"user registration failed",
			zap.Error(createErr),
		)

		return entity.User{}, fmt.Errorf("auth usecase - register: %w", createErr)
	}

	log.Info(
		"user registered",
		zap.String("user_id", createdUser.ID.String()),
		zap.String("role", string(createdUser.Role)),
	)

	return createdUser, nil
}

const maxPasswordBytes = 72

func validatePassword(password string) error {
	if password == "" {
		return errs.ErrPasswordRequired
	}
	if len(password) > maxPasswordBytes {
		return errs.ErrPasswordTooLong
	}

	return nil
}

func (service *authService) Login(
	ctx context.Context,
	email string,
	password string,
) (port.AuthTokens, error) {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("auth_usecase")

	email = strings.TrimSpace(strings.ToLower(email))

	authUser, err := service.userRepository.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			return port.AuthTokens{}, errs.ErrInvalidCredentials
		}

		log.Error(
			"login failed",
			zap.Error(err),
		)

		return port.AuthTokens{}, fmt.Errorf("auth usecase - login: %w", err)
	}

	if err = service.passwordHasher.Compare(authUser.PasswordHash, password); err != nil {
		if errors.Is(err, errs.ErrInvalidCredentials) {
			return port.AuthTokens{}, errs.ErrInvalidCredentials
		}

		log.Error(
			"login failed",
			zap.Error(err),
		)

		return port.AuthTokens{}, fmt.Errorf("auth usecase - login: %w", err)
	}

	sessionID := uuid.New()

	refreshToken, refreshHash, refreshTokenErr := service.refreshTokenManager.CreateRefreshToken(sessionID)
	if refreshTokenErr != nil {
		log.Error(
			"login failed",
			zap.Error(refreshTokenErr),
		)

		return port.AuthTokens{}, fmt.Errorf("auth usecase - login: %w", refreshTokenErr)
	}

	accessToken, accessTokenErr := service.accessTokenManager.CreateAccessToken(entity.Identity{
		UserID:    authUser.User.ID,
		SessionID: sessionID,
		Role:      authUser.User.Role,
	})
	if accessTokenErr != nil {
		log.Error(
			"login failed",
			zap.Error(accessTokenErr),
		)

		return port.AuthTokens{}, fmt.Errorf("auth usecase - login: %w", accessTokenErr)
	}

	session := entity.Session{
		ID:               sessionID,
		UserID:           authUser.User.ID,
		RefreshTokenHash: refreshHash,
		ExpiresAt:        time.Now().UTC().Add(service.sessionTTL),
	}

	if err = session.Validate(); err != nil {
		log.Error(
			"login failed",
			zap.Error(err),
		)

		return port.AuthTokens{}, fmt.Errorf("auth usecase - login: validation error: %w", err)
	}

	_, err = service.sessionRepository.Create(ctx, session)
	if err != nil {
		log.Error(
			"login failed",
			zap.Error(err),
		)

		return port.AuthTokens{}, fmt.Errorf("auth usecase - login: %w", err)
	}

	return port.AuthTokens{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: session.ExpiresAt,
	}, nil
}

func (service *authService) Refresh(
	ctx context.Context,
	rawRefreshToken string,
) (port.AuthTokens, error) {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("auth_usecase")

	sessionID, oldRefreshTokenHash, err := service.refreshTokenManager.ParseRefreshToken(rawRefreshToken)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidRefreshToken) {
			return port.AuthTokens{}, errs.ErrInvalidRefreshToken
		}

		log.Error("token refresh failed", zap.Error(err))
		return port.AuthTokens{}, fmt.Errorf("auth usecase - refresh: parse refresh token: %w", err)
	}

	session, err := service.sessionRepository.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, errs.ErrSessionNotFound) {
			return port.AuthTokens{}, errs.ErrInvalidRefreshToken
		}

		log.Error("token refresh failed", zap.Error(err))
		return port.AuthTokens{}, fmt.Errorf("auth usecase - refresh: get session: %w", err)
	}

	if session.IsRevoked() || session.IsExpired(time.Now().UTC()) {
		return port.AuthTokens{}, errs.ErrInvalidRefreshToken
	}

	user, err := service.userRepository.GetByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			return port.AuthTokens{}, errs.ErrInvalidRefreshToken
		}

		log.Error("token refresh failed", zap.Error(err))
		return port.AuthTokens{}, fmt.Errorf("auth usecase - refresh: get user: %w", err)
	}

	newRefreshToken, newRefreshTokenHash, err := service.refreshTokenManager.CreateRefreshToken(session.ID)
	if err != nil {
		log.Error("token refresh failed", zap.Error(err))
		return port.AuthTokens{}, fmt.Errorf("auth usecase - refresh: create refresh token: %w", err)
	}

	accessToken, err := service.accessTokenManager.CreateAccessToken(entity.Identity{
		UserID:    session.UserID,
		SessionID: session.ID,
		Role:      user.Role,
	})
	if err != nil {
		log.Error("token refresh failed", zap.Error(err))
		return port.AuthTokens{}, fmt.Errorf("auth usecase - refresh: create access token: %w", err)
	}

	_, err = service.sessionRepository.RotateRefreshToken(
		ctx,
		session.ID,
		oldRefreshTokenHash,
		newRefreshTokenHash,
	)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidRefreshToken) {
			return port.AuthTokens{}, errs.ErrInvalidRefreshToken
		}

		log.Error("token refresh failed", zap.Error(err))
		return port.AuthTokens{}, fmt.Errorf("auth usecase - refresh: rotate refresh token: %w", err)
	}

	log.Info(
		"tokens refreshed",
		zap.String("user_id", session.UserID.String()),
		zap.String("session_id", session.ID.String()),
	)

	return port.AuthTokens{
		AccessToken:           accessToken,
		RefreshToken:          newRefreshToken,
		RefreshTokenExpiresAt: session.ExpiresAt,
	}, nil
}

func (service *authService) Logout(
	ctx context.Context,
	sessionID uuid.UUID,
) error {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("auth_usecase")

	if sessionID == uuid.Nil {
		return fmt.Errorf("auth usecase - logout: validation error: %w", errs.ErrSessionIDRequired)
	}

	if err := service.sessionRepository.Revoke(ctx, sessionID, time.Now().UTC()); err != nil {
		log.Error(
			"logout failed",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)

		return fmt.Errorf("auth usecase - logout: %w", err)
	}

	log.Info(
		"session revoked",
		zap.String("session_id", sessionID.String()),
	)

	return nil
}

func (service *authService) LogoutAll(
	ctx context.Context,
	userID uuid.UUID,
) error {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("auth_usecase")

	if userID == uuid.Nil {
		return fmt.Errorf("auth usecase - logout all: validation error: %w", errs.ErrIdentityUserIDRequired)
	}

	if err := service.sessionRepository.RevokeAllByUser(ctx, userID, time.Now().UTC()); err != nil {
		log.Error(
			"logout all failed",
			zap.String("user_id", userID.String()),
			zap.Error(err),
		)

		return fmt.Errorf("auth usecase - logout all: %w", err)
	}

	log.Info(
		"all user sessions revoked",
		zap.String("user_id", userID.String()),
	)

	return nil
}

var (
	dummyAdminID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	dummyUserID  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

func (service *authService) DummyLogin(
	ctx context.Context,
	role entity.UserRole,
) (string, error) {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("auth_usecase")

	if !role.IsValid() {
		return "", fmt.Errorf("auth usecase - dummyLogin: validation error: %w", errs.ErrUserRoleInvalid)
	}

	var id uuid.UUID

	switch role {
	case entity.UserRoleAdmin:
		id = dummyAdminID
	case entity.UserRoleUser:
		id = dummyUserID
	}

	token, generateErr := service.accessTokenManager.CreateAccessToken(entity.Identity{
		UserID:    id,
		SessionID: uuid.New(),
		Role:      role,
	})
	if generateErr != nil {
		log.Error(
			"dummy login failed",
			zap.Error(generateErr),
		)

		return "", fmt.Errorf("auth usecase - dummyLogin: %w", generateErr)
	}

	return token, nil
}
