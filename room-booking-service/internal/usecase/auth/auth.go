package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
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
		logger:         logger.Named("auth_usecase"),
	}
}

func (service *authService) Register(
	ctx context.Context,
	email string,
	password string,
	role entity.UserRole,
) (entity.User, error) {
	user := entity.User{
		Email: email,
		Role:  role,
	}
	if err := user.Validate(); err != nil {
		return entity.User{}, fmt.Errorf("auth usecase - register: validation error: %w", err)
	}

	passwordHash, err := service.passwordHasher.Hash(password)
	if err != nil {
		service.logger.Error(
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

		service.logger.Error(
			"user registration failed",
			zap.Error(createErr),
		)

		return entity.User{}, fmt.Errorf("auth usecase - register: %w", createErr)
	}

	service.logger.Info(
		"user registered",
		zap.String("user_id", createdUser.ID.String()),
		zap.String("role", string(createdUser.Role)),
	)

	return createdUser, nil
}

func (service *authService) Login(
	ctx context.Context,
	email string,
	password string,
) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	authUser, err := service.userRepository.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			return "", errs.ErrInvalidCredentials
		}

		service.logger.Error(
			"login failed",
			zap.Error(err),
		)

		return "", fmt.Errorf("auth usecase - login: %w", err)
	}

	if err = service.passwordHasher.Compare(authUser.PasswordHash, password); err != nil {
		if errors.Is(err, errs.ErrInvalidCredentials) {
			return "", errs.ErrInvalidCredentials
		}

		service.logger.Error(
			"login failed",
			zap.Error(err),
		)

		return "", fmt.Errorf("auth usecase - login: %w", err)
	}

	token, generateErr := service.tokenIssuer.Generate(authUser.User.ID, authUser.User.Role)
	if generateErr != nil {
		service.logger.Error(
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
		service.logger.Error(
			"dummy login failed",
			zap.Error(generateErr),
		)

		return "", fmt.Errorf("auth usecase - dummyLogin: %w", generateErr)
	}

	return token, nil
}
