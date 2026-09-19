package port

import "github.com/google/uuid"

type CreateBookingParams struct {
	SlotID               uuid.UUID
	CreateConferenceLink bool
}
