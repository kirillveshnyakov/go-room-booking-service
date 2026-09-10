package mapper

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
)

func CreateBookingRequestToParams(
	request dto.CreateBookingRequest,
	userID uuid.UUID,
) (port.CreateBookingParams, error) {
	slotID, err := uuid.Parse(request.SlotID)
	if err != nil {
		return port.CreateBookingParams{}, fmt.Errorf("booking mapper - parse slot ID: %w", err)
	}

	return port.CreateBookingParams{
		UserID:               userID,
		SlotID:               slotID,
		CreateConferenceLink: request.CreateConferenceLink,
	}, nil
}

func CancelBookingURIToBookingID(uri dto.CancelBookingURI) (uuid.UUID, error) {
	id, err := uuid.Parse(uri.BookingID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("booking mapper - parse booking ID: %w", err)
	}

	return id, nil
}

func BookingToResponse(booking entity.Booking) dto.BookingResponse {
	return dto.BookingResponse{
		ID:             booking.ID.String(),
		SlotID:         booking.SlotID.String(),
		UserID:         booking.UserID.String(),
		Status:         string(booking.Status),
		ConferenceLink: optionalString(booking.ConferenceLink),
		CreatedAt:      optionalUTCTime(booking.CreatedAt),
	}
}

func BookingsToResponse(bookings []entity.Booking) []dto.BookingResponse {
	result := make([]dto.BookingResponse, 0, len(bookings))
	for _, booking := range bookings {
		result = append(result, BookingToResponse(booking))
	}

	return result
}
