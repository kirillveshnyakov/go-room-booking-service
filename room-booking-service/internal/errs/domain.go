package errs

import "errors"

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrSlotNotFound     = errors.New("slot not found")
	ErrRoomNotFound     = errors.New("room not found")
	ErrBookingNotFound  = errors.New("booking not found")
	ErrScheduleNotFound = errors.New("schedule not found")

	ErrUserEmailAlreadyExists = errors.New("user email already exists")
	ErrScheduleAlreadyExists  = errors.New("schedule already exists")
	ErrRoomNameAlreadyExists  = errors.New("room name already exists")

	ErrSlotAlreadyBooked = errors.New("slot already booked")
	ErrSlotInPast        = errors.New("slot is in past")

	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)
