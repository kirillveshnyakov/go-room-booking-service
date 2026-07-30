package entity

import (
	"strings"
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

func (r Room) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errs.ErrRoomNameRequired
	}
	if r.Capacity <= 0 {
		return errs.ErrRoomCapacityInvalid
	}
	return nil
}
