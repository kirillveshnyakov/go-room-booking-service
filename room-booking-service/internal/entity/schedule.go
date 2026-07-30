package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
)

type Schedule struct {
	ID        uuid.UUID
	RoomID    uuid.UUID
	Rules     []ScheduleRule
	CreatedAt time.Time
}

type ScheduleRule struct {
	DayOfWeek DayOfWeek
	StartTime time.Duration
	EndTime   time.Duration
}

func (s ScheduleRule) Validate() error {
	if !s.DayOfWeek.IsValid() {
		return errs.ErrScheduleRuleDayInvalid
	}
	if s.StartTime < 0 || s.StartTime >= 24*time.Hour {
		return errs.ErrScheduleRuleStartTimeInvalid
	}
	if s.EndTime <= 0 || s.EndTime > 24*time.Hour {
		return errs.ErrScheduleRuleEndTimeInvalid
	}
	if s.StartTime >= s.EndTime {
		return errs.ErrScheduleRuleTimeRangeInvalid
	}
	return nil
}

func (s Schedule) Validate() error {
	if s.RoomID == uuid.Nil {
		return errs.ErrScheduleRoomIDRequired
	}
	if len(s.Rules) == 0 {
		return errs.ErrScheduleRulesRequired
	}
	seen := make(map[DayOfWeek]struct{}, len(s.Rules))

	for i, rule := range s.Rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("validate rule %d: %w", i, err)
		}

		if _, exists := seen[rule.DayOfWeek]; exists {
			return errs.ErrScheduleDuplicateDay
		}

		seen[rule.DayOfWeek] = struct{}{}
	}
	return nil
}
