package mapper

import (
	"fmt"
	"time"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

func ListFreeSlotsQueryToDate(query dto.ListFreeSlotsQuery) (time.Time, error) {
	date, err := time.Parse("2006-01-02", query.Date)
	if err != nil {
		return time.Time{}, fmt.Errorf("slot mapper - parse date: %w", err)
	}

	return date.UTC(), nil
}

func SlotToResponse(slot entity.Slot) dto.SlotResponse {
	return dto.SlotResponse{
		ID:      slot.ID.String(),
		RoomID:  slot.RoomID.String(),
		StartAt: slot.StartAt.UTC(),
		EndAt:   slot.EndAt.UTC(),
	}
}

func SlotsToResponse(slots []entity.Slot) []dto.SlotResponse {
	result := make([]dto.SlotResponse, 0, len(slots))
	for _, slot := range slots {
		result = append(result, SlotToResponse(slot))
	}

	return result
}
