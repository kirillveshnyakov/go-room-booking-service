package slot

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/tests/util/testdb"
	"github.com/stretchr/testify/require"
)

func TestSlotRepository_CreateIdempotent(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewSlotRepository(db.Pool)
	roomID := db.Room()
	start := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Minute)
	require.NoError(t, repo.Create(ctx, roomID, []time.Time{start, start.Add(30 * time.Minute)}))
	require.NoError(t, repo.Create(ctx, roomID, []time.Time{start, start.Add(30 * time.Minute)}))

	var count int
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM slots WHERE room_id = $1", roomID).Scan(&count))
	require.Equal(t, 2, count)
}

func TestSlotRepository_CreateOverlapAndRoomFK(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewSlotRepository(db.Pool)
	roomID := db.Room()
	start := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Minute)
	db.Slot(roomID, start)

	err := repo.Create(ctx, roomID, []time.Time{start.Add(15 * time.Minute)})
	require.ErrorIs(t, err, errs.ErrSlotOverlap)
	err = repo.Create(ctx, uuid.New(), []time.Time{start})
	require.ErrorIs(t, err, errs.ErrRoomNotFound)
}

func TestSlotRepository_ListFreeAndDateFilter(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewSlotRepository(db.Pool)
	roomID := db.Room()
	otherRoomID := db.Room()
	userID := db.User()
	tomorrow := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	db.Slot(roomID, tomorrow.Add(-24*time.Hour))
	active := db.Slot(roomID, tomorrow.Add(9*time.Hour))
	cancelled := db.Slot(roomID, tomorrow.Add(10*time.Hour))
	free := db.Slot(roomID, tomorrow.Add(11*time.Hour))
	db.Slot(otherRoomID, tomorrow.Add(12*time.Hour))
	db.Slot(roomID, tomorrow.Add(24*time.Hour+9*time.Hour))
	db.Booking(active, userID, "active", time.Now())
	db.Booking(cancelled, userID, "cancelled", time.Now())

	exists, err := repo.CheckExistsForDate(ctx, roomID, tomorrow)
	require.NoError(t, err)
	require.True(t, exists)
	// UTC date, not the local calendar date of the supplied time.Time.
	target := tomorrow.Add(22 * time.Hour).In(time.FixedZone("UTC+3", 3*60*60))
	got, err := repo.ListFree(ctx, roomID, target)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, []uuid.UUID{cancelled, free}, []uuid.UUID{got[0].ID, got[1].ID})

	exists, err = repo.CheckExistsForDate(ctx, roomID, tomorrow.Add(48*time.Hour))
	require.NoError(t, err)
	require.False(t, exists)
	pastSlots, err := repo.ListFree(ctx, roomID, tomorrow.Add(-24*time.Hour))
	require.NoError(t, err)
	require.Empty(t, pastSlots)
}
