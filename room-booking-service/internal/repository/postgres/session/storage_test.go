package session

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/tests/util/testdb"
	"github.com/stretchr/testify/require"
)

func TestSessionRepository_CreateUserFK(t *testing.T) {
	db := testdb.New(t)
	repo := NewSessionRepository(db.Pool)
	_, err := repo.Create(context.Background(), entity.Session{
		ID: uuid.New(), UserID: uuid.New(),
		RefreshTokenHash: bytes.Repeat([]byte{1}, 32),
		ExpiresAt:        time.Now().UTC().Add(time.Hour),
	})
	require.ErrorIs(t, err, errs.ErrUserNotFound)
}

func TestSessionRepository_RotateRefreshToken(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewSessionRepository(db.Pool)
	userID := db.User()
	oldHash := bytes.Repeat([]byte{1}, 32)
	newHash := bytes.Repeat([]byte{2}, 32)
	expiresAt := time.Now().UTC().Add(time.Hour)
	created, err := repo.Create(ctx, entity.Session{
		ID: uuid.New(), UserID: userID, RefreshTokenHash: oldHash, ExpiresAt: expiresAt,
	})
	require.NoError(t, err)
	require.Equal(t, oldHash, created.RefreshTokenHash)
	require.Nil(t, created.RevokedAt)
	require.False(t, created.CreatedAt.IsZero())
	require.False(t, created.UpdatedAt.IsZero())

	time.Sleep(2 * time.Millisecond)
	rotated, err := repo.RotateRefreshToken(ctx, created.ID, oldHash, newHash)
	require.NoError(t, err)
	require.Equal(t, newHash, rotated.RefreshTokenHash)
	require.Equal(t, expiresAt.Unix(), rotated.ExpiresAt.Unix())
	require.True(t, rotated.UpdatedAt.After(created.UpdatedAt), "rotation must update updated_at")

	_, err = repo.RotateRefreshToken(ctx, created.ID, oldHash, oldHash)
	require.ErrorIs(t, err, errs.ErrInvalidRefreshToken)
	stored, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, newHash, stored.RefreshTokenHash)

	revokedAt := time.Now().UTC()
	require.NoError(t, repo.Revoke(ctx, created.ID, revokedAt))
	_, err = repo.RotateRefreshToken(ctx, created.ID, newHash, oldHash)
	require.ErrorIs(t, err, errs.ErrInvalidRefreshToken)
	stored, err = repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.RevokedAt)
	require.Equal(t, revokedAt.Unix(), stored.RevokedAt.Unix())

	expired, err := repo.Create(ctx, entity.Session{
		ID: uuid.New(), UserID: userID, RefreshTokenHash: oldHash,
		ExpiresAt: time.Now().UTC().Add(-time.Hour),
	})
	require.NoError(t, err)
	_, err = repo.RotateRefreshToken(ctx, expired.ID, oldHash, newHash)
	require.ErrorIs(t, err, errs.ErrInvalidRefreshToken)
}

func TestSessionRepository_RevokeAllByUser(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewSessionRepository(db.Pool)
	userID, otherID := db.User(), db.User()
	hash := bytes.Repeat([]byte{1}, 32)
	create := func(userID uuid.UUID) uuid.UUID {
		t.Helper()
		id := uuid.New()
		_, err := repo.Create(ctx, entity.Session{
			ID: id, UserID: userID, RefreshTokenHash: hash,
			ExpiresAt: time.Now().UTC().Add(time.Hour),
		})
		require.NoError(t, err)
		return id
	}
	first, second, other := create(userID), create(userID), create(otherID)
	revokedAt := time.Now().UTC().Truncate(time.Microsecond)
	require.NoError(t, repo.RevokeAllByUser(ctx, userID, revokedAt))
	require.NoError(t, repo.RevokeAllByUser(ctx, userID, revokedAt.Add(time.Minute)))
	for _, id := range []uuid.UUID{first, second} {
		got, err := repo.GetByID(ctx, id)
		require.NoError(t, err)
		require.NotNil(t, got.RevokedAt)
		require.True(t, got.RevokedAt.Equal(revokedAt))
	}
	got, err := repo.GetByID(ctx, other)
	require.NoError(t, err)
	require.Nil(t, got.RevokedAt)
}
