//go:build repository_postgres

package room

import (
	"context"
	"testing"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/testdb"
	"github.com/stretchr/testify/require"
)

func TestRoomRepository_CreateConstraints(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewRoomRepository(db.Pool)
	created, err := repo.Create(ctx, entity.Room{Name: "Integration room", Capacity: 10})
	require.NoError(t, err)
	require.Equal(t, "Integration room", created.Name)
	require.Equal(t, 10, created.Capacity)

	_, err = repo.Create(ctx, entity.Room{Name: "Integration room", Capacity: 10})
	require.ErrorIs(t, err, errs.ErrRoomNameAlreadyExists)
	_, err = repo.Create(ctx, entity.Room{Name: "Invalid capacity", Capacity: 0})
	require.ErrorIs(t, err, errs.ErrRoomCapacityInvalid)
}
