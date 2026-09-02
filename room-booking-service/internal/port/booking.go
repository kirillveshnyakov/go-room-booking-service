package port

import "github.com/google/uuid"

type CreateBookingParams struct {
	UserID               uuid.UUID
	SlotID               uuid.UUID
	CreateConferenceLink bool
}
