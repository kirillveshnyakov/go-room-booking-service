package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
)

type Slot struct {
	ID      uuid.UUID
	RoomID  uuid.UUID
	StartAt time.Time
	EndAt   time.Time
}

func (s *Slot) Normalize() {
	s.StartAt = normalizeTime(s.StartAt)
	s.EndAt = normalizeTime(s.EndAt)
}

func (s *Slot) Validate() error {
	s.Normalize()

	if s.RoomID == uuid.Nil {
		return errs.ErrSlotRoomIDRequired
	}
	if s.StartAt.IsZero() {
		return errs.ErrSlotStartAtRequired
	}
	if s.EndAt.IsZero() {
		return errs.ErrSlotEndAtRequired
	}
	if !s.EndAt.After(s.StartAt) {
		return errs.ErrSlotTimeRangeInvalid
	}
	if s.EndAt.Sub(s.StartAt) != 30*time.Minute {
		return errs.ErrSlotDurationInvalid
	}
	return nil
}
