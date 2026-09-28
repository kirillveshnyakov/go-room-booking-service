package slot

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/usecase/slot/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestSlotService_ListFree(t *testing.T) {
	user := entity.Identity{UserID: uuid.New(), SessionID: uuid.New(), Role: entity.UserRoleUser}
	admin := user
	admin.Role = entity.UserRoleAdmin
	roomID := uuid.New()
	date := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	rule := entity.ScheduleRule{DayOfWeek: entity.GetDayOfWeek(date), StartTime: 9 * time.Hour, EndTime: 10 * time.Hour}
	starts := []time.Time{date.Add(9 * time.Hour), date.Add(9*time.Hour + 30*time.Minute)}
	slots := []entity.Slot{{ID: uuid.New(), RoomID: roomID, StartAt: starts[0], EndAt: starts[1]}}
	technicalErr := errors.New("database unavailable")

	expectRoom := func(rooms *mocks.MockroomRepository) {
		rooms.EXPECT().GetByID(gomock.Any(), roomID).Return(entity.Room{ID: roomID}, nil)
	}
	expectExistingSlots := func(repo *mocks.MockslotRepository) {
		repo.EXPECT().CheckExistsForDate(gomock.Any(), roomID, date).Return(true, nil)
	}
	expectMissingSlots := func(repo *mocks.MockslotRepository, rooms *mocks.MockroomRepository) {
		expectRoom(rooms)
		repo.EXPECT().CheckExistsForDate(gomock.Any(), roomID, date).Return(false, nil)
	}

	tests := []struct {
		name       string
		actor      entity.Identity
		date       time.Time
		setupMocks func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository)
		wantList   []entity.Slot
		wantErr    error
	}{
		{
			name:  "existing slots",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectRoom(rooms)
				expectExistingSlots(repo)
				repo.EXPECT().ListFree(gomock.Any(), roomID, date).Return(slots, nil)
			},
			wantList: slots,
		},
		{
			name:  "admin allowed",
			actor: admin,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectRoom(rooms)
				expectExistingSlots(repo)
				repo.EXPECT().ListFree(gomock.Any(), roomID, date).Return(slots, nil)
			},
			wantList: slots,
		},
		{
			name:  "empty result",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectRoom(rooms)
				expectExistingSlots(repo)
				repo.EXPECT().ListFree(gomock.Any(), roomID, date).Return([]entity.Slot{}, nil)
			},
			wantList: []entity.Slot{},
		},
		{
			name:    "invalid identity",
			date:    date,
			wantErr: errs.ErrForbidden,
		},
		{
			name:    "missing session ID",
			actor:   entity.Identity{UserID: user.UserID, Role: entity.UserRoleUser},
			date:    date,
			wantErr: errs.ErrForbidden,
		},
		{
			name:    "invalid role",
			actor:   entity.Identity{UserID: user.UserID, SessionID: user.SessionID, Role: "owner"},
			date:    date,
			wantErr: errs.ErrForbidden,
		},
		{
			name:  "room not found",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				rooms.EXPECT().GetByID(gomock.Any(), roomID).Return(entity.Room{}, errs.ErrRoomNotFound)
			},
			wantErr: errs.ErrRoomNotFound,
		},
		{
			name:  "wrapped room not found",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				rooms.EXPECT().GetByID(gomock.Any(), roomID).Return(entity.Room{}, fmt.Errorf("lookup: %w", errs.ErrRoomNotFound))
			},
			wantErr: errs.ErrRoomNotFound,
		},
		{
			name:  "room technical error",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				rooms.EXPECT().GetByID(gomock.Any(), roomID).Return(entity.Room{}, technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name:  "existence check error",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectRoom(rooms)
				repo.EXPECT().CheckExistsForDate(gomock.Any(), roomID, date).Return(false, technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name:  "schedule missing",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectMissingSlots(repo, rooms)
				schedules.EXPECT().GetRuleForDay(gomock.Any(), roomID, rule.DayOfWeek).Return(entity.ScheduleRule{}, errs.ErrScheduleNotFound)
			},
			wantList: []entity.Slot{},
		},
		{
			name:  "rule missing",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectMissingSlots(repo, rooms)
				schedules.EXPECT().GetRuleForDay(gomock.Any(), roomID, rule.DayOfWeek).Return(entity.ScheduleRule{}, errs.ErrScheduleRuleNotFound)
			},
			wantList: []entity.Slot{},
		},
		{
			name:  "wrapped missing rule",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectMissingSlots(repo, rooms)
				schedules.EXPECT().GetRuleForDay(gomock.Any(), roomID, rule.DayOfWeek).Return(entity.ScheduleRule{}, fmt.Errorf("lookup: %w", errs.ErrScheduleRuleNotFound))
			},
			wantList: []entity.Slot{},
		},
		{
			name:  "rule technical error",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectMissingSlots(repo, rooms)
				schedules.EXPECT().GetRuleForDay(gomock.Any(), roomID, rule.DayOfWeek).Return(entity.ScheduleRule{}, technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name:  "generate two slots",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectMissingSlots(repo, rooms)
				schedules.EXPECT().GetRuleForDay(gomock.Any(), roomID, rule.DayOfWeek).Return(rule, nil)
				create := repo.EXPECT().Create(gomock.Any(), roomID, starts).Return(nil)
				repo.EXPECT().ListFree(gomock.Any(), roomID, date).Return(slots, nil).After(create)
			},
			wantList: slots,
		},
		{
			name:  "skip incomplete final slot",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectMissingSlots(repo, rooms)
				schedules.EXPECT().GetRuleForDay(gomock.Any(), roomID, rule.DayOfWeek).Return(entity.ScheduleRule{StartTime: 9 * time.Hour, EndTime: 10*time.Hour + 15*time.Minute}, nil)
				create := repo.EXPECT().Create(gomock.Any(), roomID, starts).Return(nil)
				repo.EXPECT().ListFree(gomock.Any(), roomID, date).Return(slots, nil).After(create)
			},
			wantList: slots,
		},
		{
			name:  "exactly one slot",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectMissingSlots(repo, rooms)
				schedules.EXPECT().GetRuleForDay(gomock.Any(), roomID, rule.DayOfWeek).Return(entity.ScheduleRule{StartTime: 9 * time.Hour, EndTime: 9*time.Hour + 30*time.Minute}, nil)
				create := repo.EXPECT().Create(gomock.Any(), roomID, starts[:1]).Return(nil)
				repo.EXPECT().ListFree(gomock.Any(), roomID, date).Return(slots[:1], nil).After(create)
			},
			wantList: slots[:1],
		},
		{
			name:  "interval shorter than slot",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectMissingSlots(repo, rooms)
				schedules.EXPECT().GetRuleForDay(gomock.Any(), roomID, rule.DayOfWeek).Return(entity.ScheduleRule{StartTime: 9 * time.Hour, EndTime: 9*time.Hour + 29*time.Minute}, nil)
				repo.EXPECT().ListFree(gomock.Any(), roomID, date).Return([]entity.Slot{}, nil)
			},
			wantList: []entity.Slot{},
		},
		{
			name:  "UTC date crosses local day",
			actor: user,
			date:  date.Add(22 * time.Hour).In(time.FixedZone("UTC+3", 3*60*60)),
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectRoom(rooms)
				repo.EXPECT().CheckExistsForDate(gomock.Any(), roomID, date.Add(22*time.Hour)).Return(false, nil)
				schedules.EXPECT().GetRuleForDay(gomock.Any(), roomID, rule.DayOfWeek).Return(rule, nil)
				create := repo.EXPECT().Create(gomock.Any(), roomID, starts).Return(nil)
				repo.EXPECT().ListFree(gomock.Any(), roomID, date.Add(22*time.Hour)).Return(slots, nil).After(create)
			},
			wantList: slots,
		},
		{
			name:  "generation error",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectMissingSlots(repo, rooms)
				schedules.EXPECT().GetRuleForDay(gomock.Any(), roomID, rule.DayOfWeek).Return(rule, nil)
				repo.EXPECT().Create(gomock.Any(), roomID, starts).Return(technicalErr)
			},
			wantErr: technicalErr,
		},
		{
			name:  "list error after generation",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectMissingSlots(repo, rooms)
				schedules.EXPECT().GetRuleForDay(gomock.Any(), roomID, rule.DayOfWeek).Return(rule, nil)
				create := repo.EXPECT().Create(gomock.Any(), roomID, starts).Return(nil)
				repo.EXPECT().ListFree(gomock.Any(), roomID, date).Return(nil, technicalErr).After(create)
			},
			wantErr: technicalErr,
		},
		{
			name:  "list error for existing slots",
			actor: user,
			date:  date,
			setupMocks: func(repo *mocks.MockslotRepository, schedules *mocks.MockscheduleRepository, rooms *mocks.MockroomRepository) {
				expectRoom(rooms)
				expectExistingSlots(repo)
				repo.EXPECT().ListFree(gomock.Any(), roomID, date).Return(nil, technicalErr)
			},
			wantErr: technicalErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockslotRepository(ctrl)
			schedules := mocks.NewMockscheduleRepository(ctrl)
			rooms := mocks.NewMockroomRepository(ctrl)
			if tt.setupMocks != nil {
				tt.setupMocks(repo, schedules, rooms)
			}
			service := NewSlotService(repo, schedules, rooms, zap.NewNop())
			got, err := service.ListFree(context.Background(), tt.actor, roomID, tt.date)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantList, got)
		})
	}
}
