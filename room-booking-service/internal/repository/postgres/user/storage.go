package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres/transactor"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/sqlcgen"
)

const (
	userEmailUniqueConstraint = "users_email_unique"
)

type userRepository struct {
	queries *sqlcgen.Queries
}

func NewUserRepository(db sqlcgen.DBTX) *userRepository {
	return &userRepository{
		queries: sqlcgen.New(db),
	}
}

func (repo *userRepository) getQueries(ctx context.Context) *sqlcgen.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return repo.queries.WithTx(tx)
	}

	return repo.queries
}

func (repo *userRepository) Create(
	ctx context.Context,
	email string,
	role entity.UserRole,
	passwordHash string,
) (entity.User, error) {
	pgRole, roleErr := roleToPG(role)
	if roleErr != nil {
		return entity.User{}, fmt.Errorf("user repository - create: %w", roleErr)
	}

	user, err := repo.getQueries(ctx).CreateUser(ctx, sqlcgen.CreateUserParams{
		Email:        email,
		Role:         pgRole,
		PasswordHash: passwordHash,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == pgerrcode.UniqueViolation &&
			pgErr.ConstraintName == userEmailUniqueConstraint {
			return entity.User{}, errs.ErrUserEmailAlreadyExists
		}

		return entity.User{}, fmt.Errorf("user repository - create: %w", err)
	}

	return entity.User{
		ID:        user.ID,
		Email:     user.Email,
		Role:      entity.UserRole(user.Role),
		CreatedAt: user.CreatedAt.Time.UTC(),
	}, nil
}

func (repo *userRepository) GetByEmail(
	ctx context.Context,
	email string,
) (entity.AuthUser, error) {
	user, err := repo.getQueries(ctx).GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.AuthUser{}, errs.ErrUserNotFound
		}

		return entity.AuthUser{}, fmt.Errorf("user repository - get by email: %w", err)
	}

	return entity.AuthUser{
		User: entity.User{
			ID:        user.ID,
			Email:     user.Email,
			Role:      entity.UserRole(user.Role),
			CreatedAt: user.CreatedAt.Time.UTC(),
		},
		PasswordHash: user.PasswordHash,
	}, nil
}

func (repo *userRepository) GetByID(
	ctx context.Context,
	userID uuid.UUID,
) (entity.User, error) {
	user, err := repo.getQueries(ctx).GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.User{}, errs.ErrUserNotFound
		}

		return entity.User{}, fmt.Errorf("user repository - get by id: %w", err)
	}

	return entity.User{
		ID:        user.ID,
		Email:     user.Email,
		Role:      entity.UserRole(user.Role),
		CreatedAt: user.CreatedAt.Time.UTC(),
	}, nil
}

func roleToPG(role entity.UserRole) (sqlcgen.UserRole, error) {
	switch role {
	case entity.UserRoleAdmin:
		return sqlcgen.UserRoleAdmin, nil
	case entity.UserRoleUser:
		return sqlcgen.UserRoleUser, nil
	default:
		return sqlcgen.UserRoleUser, fmt.Errorf("unknown role - %s", role)
	}
}
