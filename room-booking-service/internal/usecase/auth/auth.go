package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
	"go.uber.org/zap"
)

type (
	userRepository interface {
		Create(ctx context.Context, email string, role entity.UserRole, passwordHash string) (entity.User, error)
		GetByEmail(ctx context.Context, email string) (entity.AuthUser, error)
	}

	passwordHasher interface {
		Hash(password string) (string, error)
		Compare(hash, password string) error
	}

	tokenIssuer interface {
		Generate(userID uuid.UUID, role entity.UserRole) (string, error)
	}
)

type authService struct {
	userRepository userRepository
	passwordHasher passwordHasher
	tokenIssuer    tokenIssuer
	logger         *zap.Logger
}

func NewAuthService(
	userRepository userRepository,
	passwordHasher passwordHasher,
	tokenIssuer tokenIssuer,
	logger *zap.Logger,
) *authService {
	return &authService{
		userRepository: userRepository,
		passwordHasher: passwordHasher,
		tokenIssuer:    tokenIssuer,
		logger:         logger,
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
) (string, error) {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("auth_usecase")

	email = strings.TrimSpace(strings.ToLower(email))

	authUser, err := service.userRepository.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			return "", errs.ErrInvalidCredentials
		}

		log.Error(
			"login failed",
			zap.Error(err),
		)

		return "", fmt.Errorf("auth usecase - login: %w", err)
	}

	if err = service.passwordHasher.Compare(authUser.PasswordHash, password); err != nil {
		if errors.Is(err, errs.ErrInvalidCredentials) {
			return "", errs.ErrInvalidCredentials
		}

		log.Error(
			"login failed",
			zap.Error(err),
		)

		return "", fmt.Errorf("auth usecase - login: %w", err)
	}

	token, generateErr := service.tokenIssuer.Generate(authUser.User.ID, authUser.User.Role)
	if generateErr != nil {
		log.Error(
			"login failed",
			zap.Error(generateErr),
		)

		return "", fmt.Errorf("auth usecase - login: %w", generateErr)
	}

	return token, nil
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

	token, generateErr := service.tokenIssuer.Generate(id, role)
	if generateErr != nil {
		log.Error(
			"dummy login failed",
			zap.Error(generateErr),
		)

		return "", fmt.Errorf("auth usecase - dummyLogin: %w", generateErr)
	}

	return token, nil
}
