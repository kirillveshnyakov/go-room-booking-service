package mapper

import (
	"fmt"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

func ScheduleRuleRequestsToEntity(
	rules []dto.ScheduleRuleRequest,
) ([]entity.ScheduleRule, error) {
	result := make([]entity.ScheduleRule, 0, len(rules))
	for index, rule := range rules {
		startTime, err := timeStringToDuration(rule.StartTime)
		if err != nil {
			return nil, fmt.Errorf("schedule mapper - parse rule %d start time: %w", index, err)
		}

		endTime, err := timeStringToDuration(rule.EndTime)
		if err != nil {
			return nil, fmt.Errorf("schedule mapper - parse rule %d end time: %w", index, err)
		}

		result = append(result, entity.ScheduleRule{
			DayOfWeek: entity.DayOfWeek(rule.DayOfWeek),
			StartTime: startTime,
			EndTime:   endTime,
		})
	}

	return result, nil
}

func ScheduleToResponse(schedule entity.Schedule) dto.ScheduleResponse {
	return dto.ScheduleResponse{
		ID:        schedule.ID.String(),
		RoomID:    schedule.RoomID.String(),
		Rules:     ScheduleRulesToResponse(schedule.Rules),
		CreatedAt: schedule.CreatedAt.UTC(),
	}
}

func ScheduleRulesToResponse(rules []entity.ScheduleRule) []dto.ScheduleRuleResponse {
	result := make([]dto.ScheduleRuleResponse, 0, len(rules))
	for _, rule := range rules {
		result = append(result, dto.ScheduleRuleResponse{
			DayOfWeek: int(rule.DayOfWeek),
			StartTime: durationToTimeString(rule.StartTime),
			EndTime:   durationToTimeString(rule.EndTime),
		})
	}

	return result
}
