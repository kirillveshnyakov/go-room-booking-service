package dto

import "time"

type CreateBookingRequest struct {
	SlotID               string `json:"slotId" binding:"required,uuid"`
	CreateConferenceLink bool   `json:"createConferenceLink"`
}

type CreateBookingResponse struct {
	Booking BookingResponse `json:"booking"`
}

type ListBookingsQuery struct {
	Page     int `form:"page,default=1" binding:"gte=1"`
	PageSize int `form:"pageSize,default=20" binding:"gte=1,lte=100"`
}

type ListBookingsResponse struct {
	Bookings   []BookingResponse  `json:"bookings"`
	Pagination PaginationResponse `json:"pagination"`
}

type ListMyBookingsResponse struct {
	Bookings []BookingResponse `json:"bookings"`
}

type CancelBookingURI struct {
	BookingID string `uri:"bookingId" binding:"required,uuid"`
}

type CancelBookingResponse struct {
	Booking BookingResponse `json:"booking"`
}

type BookingResponse struct {
	ID             string     `json:"id"`
	SlotID         string     `json:"slotId"`
	UserID         string     `json:"userId"`
	Status         string     `json:"status"`
	ConferenceLink *string    `json:"conferenceLink,omitempty"`
	CreatedAt      *time.Time `json:"createdAt,omitempty"`
}

type PaginationResponse struct {
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}
