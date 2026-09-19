package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres/transactor"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/sqlcgen"
)

const sessionsUserIDFKConstraint = "sessions_user_id_fk"

type sessionRepository struct {
	queries *sqlcgen.Queries
}

func NewSessionRepository(db sqlcgen.DBTX) *sessionRepository {
	return &sessionRepository{
		queries: sqlcgen.New(db),
	}
}

func (repo *sessionRepository) getQueries(ctx context.Context) *sqlcgen.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return repo.queries.WithTx(tx)
	}

	return repo.queries
}

func (repo *sessionRepository) Create(
	ctx context.Context,
	session entity.Session,
) (entity.Session, error) {
	createdSession, err := repo.getQueries(ctx).CreateSession(ctx, sqlcgen.CreateSessionParams{
		ID:               session.ID,
		UserID:           session.UserID,
		RefreshTokenHash: session.RefreshTokenHash,
		ExpiresAt:        timeToPG(session.ExpiresAt),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == pgerrcode.ForeignKeyViolation &&
			pgErr.ConstraintName == sessionsUserIDFKConstraint {
			return entity.Session{}, errs.ErrUserNotFound
		}

		return entity.Session{}, fmt.Errorf("session repository - create: %w", err)
	}

	return sessionToEntity(createdSession), nil
}

func (repo *sessionRepository) GetByID(
	ctx context.Context,
	sessionID uuid.UUID,
) (entity.Session, error) {
	session, err := repo.getQueries(ctx).GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Session{}, errs.ErrSessionNotFound
		}

		return entity.Session{}, fmt.Errorf("session repository - get by id: %w", err)
	}

	return sessionToEntity(session), nil
}

func (repo *sessionRepository) Revoke(
	ctx context.Context,
	sessionID uuid.UUID,
	revokedAt time.Time,
) error {
	err := repo.getQueries(ctx).Revoke(ctx, sqlcgen.RevokeParams{
		RevokedAt: timeToPG(revokedAt),
		ID:        sessionID,
	})
	if err != nil {
		return fmt.Errorf("session repository - revoke: %w", err)
	}

	return nil
}

func (repo *sessionRepository) RevokeAllByUser(
	ctx context.Context,
	userID uuid.UUID,
	revokedAt time.Time,
) error {
	err := repo.getQueries(ctx).RevokeAllByUser(ctx, sqlcgen.RevokeAllByUserParams{
		RevokedAt: timeToPG(revokedAt),
		UserID:    userID,
	})
	if err != nil {
		return fmt.Errorf("session repository - revoke all by user: %w", err)
	}

	return nil
}

func (repo *sessionRepository) RotateRefreshToken(
	ctx context.Context,
	sessionID uuid.UUID,
	oldRefreshTokenHash []byte,
	newRefreshTokenHash []byte,
) (entity.Session, error) {
	session, err := repo.getQueries(ctx).RotateRefreshToken(ctx, sqlcgen.RotateRefreshTokenParams{
		NewRefreshTokenHash: newRefreshTokenHash,
		ID:                  sessionID,
		OldRefreshTokenHash: oldRefreshTokenHash,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Session{}, errs.ErrInvalidRefreshToken
		}

		return entity.Session{}, fmt.Errorf("session repository - rotate refresh token: %w", err)
	}

	return sessionToEntity(session), nil
}

func sessionToEntity(src sqlcgen.Session) entity.Session {
	var revokedAt *time.Time
	if src.RevokedAt.Valid {
		value := src.RevokedAt.Time.UTC()
		revokedAt = &value
	}

	return entity.Session{
		ID:               src.ID,
		UserID:           src.UserID,
		RefreshTokenHash: src.RefreshTokenHash,
		ExpiresAt:        src.ExpiresAt.Time.UTC(),
		RevokedAt:        revokedAt,
		CreatedAt:        src.CreatedAt.Time.UTC(),
		UpdatedAt:        src.UpdatedAt.Time.UTC(),
	}
}

func timeToPG(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value.UTC(),
		Valid: true,
	}
}
