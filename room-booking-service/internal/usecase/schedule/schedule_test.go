package schedule

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/usecase/schedule/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestScheduleService_Create(t *testing.T) {
	admin := entity.Identity{UserID: uuid.New(), SessionID: uuid.New(), Role: entity.UserRoleAdmin}
	user := admin
	user.Role = entity.UserRoleUser
	roomID := uuid.New()
	rule := entity.ScheduleRule{DayOfWeek: 1, StartTime: 9 * time.Hour, EndTime: 18 * time.Hour}
	rules := []entity.ScheduleRule{rule}
	repositoryErr := errors.New("database unavailable")
	tests := []struct {
		name             string
		actor            entity.Identity
		roomID           uuid.UUID
		rules            []entity.ScheduleRule
		repoErr, wantErr error
	}{
		{name: "success", actor: admin, roomID: roomID, rules: rules},
		{name: "multiple days", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{rule, {DayOfWeek: 7, StartTime: 10 * time.Hour, EndTime: 12 * time.Hour}}},
		{name: "full day boundaries", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{{DayOfWeek: 7, EndTime: 24 * time.Hour}}},
		{name: "user forbidden", actor: user, roomID: roomID, rules: rules, wantErr: errs.ErrForbidden},
		{name: "invalid identity", roomID: roomID, rules: rules, wantErr: errs.ErrForbidden},
		{name: "missing session ID", actor: entity.Identity{UserID: admin.UserID, Role: entity.UserRoleAdmin}, roomID: roomID, rules: rules, wantErr: errs.ErrForbidden},
		{name: "invalid role", actor: entity.Identity{UserID: admin.UserID, SessionID: admin.SessionID, Role: "owner"}, roomID: roomID, rules: rules, wantErr: errs.ErrForbidden},
		{name: "missing room ID", actor: admin, rules: rules, wantErr: errs.ErrScheduleRoomIDRequired},
		{name: "nil rules", actor: admin, roomID: roomID, wantErr: errs.ErrScheduleRulesRequired},
		{name: "empty rules", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{}, wantErr: errs.ErrScheduleRulesRequired},
		{name: "duplicate days", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{rule, rule}, wantErr: errs.ErrScheduleDuplicateDay},
		{name: "zero weekday", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{{DayOfWeek: 0, EndTime: time.Hour}}, wantErr: errs.ErrScheduleRuleDayInvalid},
		{name: "invalid weekday", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{{DayOfWeek: 8, EndTime: time.Hour}}, wantErr: errs.ErrScheduleRuleDayInvalid},
		{name: "negative start", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{{DayOfWeek: 1, StartTime: -1, EndTime: time.Hour}}, wantErr: errs.ErrScheduleRuleStartTimeInvalid},
		{name: "start at 24h", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{{DayOfWeek: 1, StartTime: 24 * time.Hour, EndTime: 24 * time.Hour}}, wantErr: errs.ErrScheduleRuleStartTimeInvalid},
		{name: "zero end", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{{DayOfWeek: 1}}, wantErr: errs.ErrScheduleRuleEndTimeInvalid},
		{name: "end above 24h", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{{DayOfWeek: 1, EndTime: 25 * time.Hour}}, wantErr: errs.ErrScheduleRuleEndTimeInvalid},
		{name: "equal start and end", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{{DayOfWeek: 1, StartTime: time.Hour, EndTime: time.Hour}}, wantErr: errs.ErrScheduleRuleTimeRangeInvalid},
		{name: "reversed range", actor: admin, roomID: roomID, rules: []entity.ScheduleRule{{DayOfWeek: 1, StartTime: 2 * time.Hour, EndTime: time.Hour}}, wantErr: errs.ErrScheduleRuleTimeRangeInvalid},
		{name: "schedule exists", actor: admin, roomID: roomID, rules: rules, repoErr: errs.ErrScheduleAlreadyExists, wantErr: errs.ErrScheduleAlreadyExists},
		{name: "room not found", actor: admin, roomID: roomID, rules: rules, repoErr: errs.ErrRoomNotFound, wantErr: errs.ErrRoomNotFound},
		{name: "wrapped schedule exists", actor: admin, roomID: roomID, rules: rules, repoErr: fmt.Errorf("create: %w", errs.ErrScheduleAlreadyExists), wantErr: errs.ErrScheduleAlreadyExists},
		{name: "technical error", actor: admin, roomID: roomID, rules: rules, repoErr: repositoryErr, wantErr: repositoryErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewMockscheduleRepository(gomock.NewController(t))
			expected := entity.Schedule{RoomID: tt.roomID, Rules: tt.rules}
			created := expected
			created.ID = uuid.New()
			if tt.wantErr == nil || tt.repoErr != nil {
				repo.EXPECT().Create(gomock.Any(), expected).Return(created, tt.repoErr)
			}
			got, err := NewScheduleService(repo, zap.NewNop()).Create(context.Background(), tt.actor, tt.roomID, tt.rules)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, entity.Schedule{}, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, created, got)
		})
	}
}
