package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
)

type Booking struct {
	ID             uuid.UUID
	SlotID         uuid.UUID
	UserID         uuid.UUID
	Status         BookingStatus
	ConferenceLink string
	CreatedAt      time.Time
}

func (b Booking) Validate() error {
	if b.SlotID == uuid.Nil {
		return errs.ErrBookingSlotIDRequired
	}
	if b.UserID == uuid.Nil {
		return errs.ErrBookingUserIDRequired
	}
	if !b.Status.IsValid() {
		return errs.ErrBookingStatusInvalid
	}
	return nil
}
