package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
)

type Room struct {
	ID          uuid.UUID
	Name        string
	Description string
	Capacity    int
	CreatedAt   time.Time
}

func (r *Room) Normalize() {
	r.Name = normalizeText(r.Name)
	r.Description = normalizeText(r.Description)
	r.CreatedAt = normalizeTime(r.CreatedAt)
}

func (r *Room) Validate() error {
	r.Normalize()

	if r.Name == "" {
		return errs.ErrRoomNameRequired
	}
	if r.Capacity <= 0 {
		return errs.ErrRoomCapacityInvalid
	}
	return nil
}
