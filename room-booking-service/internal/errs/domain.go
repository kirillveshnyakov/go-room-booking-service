package errs

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrSlotNotFound         = errors.New("slot not found")
	ErrRoomNotFound         = errors.New("room not found")
	ErrBookingNotFound      = errors.New("booking not found")
	ErrScheduleNotFound     = errors.New("schedule not found")
	ErrScheduleRuleNotFound = errors.New("schedule rule not found")
	ErrSessionNotFound      = errors.New("session not found")

	ErrUserEmailAlreadyExists = errors.New("user email already exists")
	ErrScheduleAlreadyExists  = errors.New("schedule already exists")
	ErrRoomNameAlreadyExists  = errors.New("room name already exists")

	ErrSlotAlreadyBooked = errors.New("slot already booked")
	ErrSlotOverlap       = errors.New("slot overlaps an existing slot")
	ErrSlotInPast        = errors.New("slot is in past")

	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidCredentials = errors.New("invalid credentials")

	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)
