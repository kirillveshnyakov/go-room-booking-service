//go:build repository_postgres

package schedule

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres/transactor"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/tests/util/testdb"
	"github.com/stretchr/testify/require"
)

func TestScheduleRepository_CreateRollsBackRules(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewScheduleRepository(db.Pool, transactor.NewTransactor(db.Pool))
	roomID := db.Room()
	rule := entity.ScheduleRule{DayOfWeek: 1, StartTime: 9 * time.Hour, EndTime: 10 * time.Hour}
	_, err := repo.Create(ctx, entity.Schedule{RoomID: roomID, Rules: []entity.ScheduleRule{rule, rule}})
	require.ErrorIs(t, err, errs.ErrScheduleDuplicateDay)

	var count int
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM schedules WHERE room_id = $1", roomID).Scan(&count))
	require.Zero(t, count, "failed rule insert must roll back the parent schedule")

	created, err := repo.Create(ctx, entity.Schedule{RoomID: roomID, Rules: []entity.ScheduleRule{rule}})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, roomID, created.RoomID)
	require.Equal(t, []entity.ScheduleRule{rule}, created.Rules)
	_, err = repo.Create(ctx, entity.Schedule{RoomID: roomID, Rules: []entity.ScheduleRule{rule}})
	require.ErrorIs(t, err, errs.ErrScheduleAlreadyExists)
}

func TestScheduleRepository_GetRuleForDayMissingStates(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	repo := NewScheduleRepository(db.Pool, transactor.NewTransactor(db.Pool))
	roomID := db.Room()
	_, err := repo.GetRuleForDay(ctx, roomID, 1)
	require.ErrorIs(t, err, errs.ErrScheduleNotFound)

	rule := entity.ScheduleRule{DayOfWeek: 2, StartTime: 9 * time.Hour, EndTime: 11 * time.Hour}
	_, err = repo.Create(ctx, entity.Schedule{RoomID: roomID, Rules: []entity.ScheduleRule{rule}})
	require.NoError(t, err)
	_, err = repo.GetRuleForDay(ctx, roomID, 1)
	require.ErrorIs(t, err, errs.ErrScheduleRuleNotFound)
	got, err := repo.GetRuleForDay(ctx, roomID, 2)
	require.NoError(t, err)
	require.Equal(t, rule, got)
}
