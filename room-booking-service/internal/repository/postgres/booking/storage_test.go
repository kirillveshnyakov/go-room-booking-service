//go:build repository_postgres

package booking

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres/transactor"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/testdb"
	"github.com/stretchr/testify/require"
)

func TestBookingRepository_CreateSlotState(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewBookingRepository(db.Pool, transactor.NewTransactor(db.Pool))
	userID := db.User()
	roomID := db.Room()
	pastSlotID := db.Slot(roomID, time.Now().UTC().Add(-time.Hour))

	for _, tt := range []struct {
		name    string
		slotID  uuid.UUID
		wantErr error
	}{
		{name: "missing slot", slotID: uuid.New(), wantErr: errs.ErrSlotNotFound},
		{name: "past slot", slotID: pastSlotID, wantErr: errs.ErrSlotInPast},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Create(ctx, tt.slotID, userID, "")
			require.ErrorIs(t, err, tt.wantErr)
			require.Equal(t, entity.Booking{}, got)
		})
	}
}

func TestBookingRepository_CreateUserFK(t *testing.T) {
	db := testdb.New(t)
	repo := NewBookingRepository(db.Pool, transactor.NewTransactor(db.Pool))
	slotID := db.Slot(db.Room(), time.Now().UTC().Add(24*time.Hour))
	got, err := repo.Create(context.Background(), slotID, uuid.New(), "")
	require.ErrorIs(t, err, errs.ErrUserNotFound)
	require.Equal(t, entity.Booking{}, got)
}

func TestBookingRepository_OneActivePerSlot(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewBookingRepository(db.Pool, transactor.NewTransactor(db.Pool))
	slotID := db.Slot(db.Room(), time.Now().UTC().Add(24*time.Hour))
	userID := db.User()

	first, err := repo.Create(ctx, slotID, userID, "https://meet.example.test/one")
	require.NoError(t, err)
	require.Equal(t, entity.BookingStatusActive, first.Status)
	require.Equal(t, "https://meet.example.test/one", first.ConferenceLink)

	_, err = repo.Create(ctx, slotID, userID, "")
	require.ErrorIs(t, err, errs.ErrSlotAlreadyBooked)

	require.NoError(t, repo.Cancel(ctx, first.ID))
	second, err := repo.Create(ctx, slotID, userID, "")
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)
	require.Equal(t, entity.BookingStatusActive, second.Status)

	_, err = repo.Create(ctx, slotID, userID, "")
	require.ErrorIs(t, err, errs.ErrSlotAlreadyBooked)
}

func TestBookingRepository_ListPagination(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewBookingRepository(db.Pool, transactor.NewTransactor(db.Pool))
	roomID, userID := db.Room(), db.User()
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	ids := make([]uuid.UUID, 3)
	for i := range ids {
		slotID := db.Slot(roomID, base.Add(time.Duration(i)*time.Hour))
		ids[i] = db.Booking(slotID, userID, "active", base.Add(time.Duration(i)*time.Minute))
	}

	first, total, err := repo.List(ctx, 2, 0)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, first, 2)
	require.Equal(t, []uuid.UUID{ids[2], ids[1]}, []uuid.UUID{first[0].ID, first[1].ID})

	second, total, err := repo.List(ctx, 2, 2)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, second, 1)
	require.Equal(t, ids[0], second[0].ID)
}

func TestBookingRepository_ListUserFuture(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewBookingRepository(db.Pool, transactor.NewTransactor(db.Pool))
	roomID, userID, otherID := db.Room(), db.User(), db.User()
	now := time.Now().UTC().Truncate(time.Second)
	past := db.Slot(roomID, now.Add(-time.Hour))
	futureLate := db.Slot(roomID, now.Add(48*time.Hour))
	futureEarly := db.Slot(roomID, now.Add(24*time.Hour))
	otherSlot := db.Slot(roomID, now.Add(72*time.Hour))
	db.Booking(past, userID, "active", now)
	lateID := db.Booking(futureLate, userID, "active", now)
	earlyID := db.Booking(futureEarly, userID, "cancelled", now)
	db.Booking(otherSlot, otherID, "active", now)

	got, err := repo.ListUserFuture(ctx, userID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, []uuid.UUID{earlyID, lateID}, []uuid.UUID{got[0].ID, got[1].ID})
	require.Equal(t, entity.BookingStatusCancelled, got[0].Status)
}
