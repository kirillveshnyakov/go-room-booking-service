package httperror

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
)

const (
	CodeInvalidRequest     = "INVALID_REQUEST"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeNotFound           = "NOT_FOUND"
	CodeRoomNotFound       = "ROOM_NOT_FOUND"
	CodeSlotNotFound       = "SLOT_NOT_FOUND"
	CodeSlotAlreadyBooked  = "SLOT_ALREADY_BOOKED"
	CodeBookingNotFound    = "BOOKING_NOT_FOUND"
	CodeForbidden          = "FORBIDDEN"
	CodeScheduleExists     = "SCHEDULE_EXISTS"
	CodeInternalError      = "INTERNAL_ERROR"
	CodeTooManyRequests    = "TOO_MANY_REQUESTS"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error Error `json:"error"`
}

func WriteError(
	c *gin.Context,
	status int,
	code string,
	message string,
) {
	c.JSON(status, ErrorResponse{
		Error: Error{
			Code:    code,
			Message: message,
		},
	})
}

func HandleError(c *gin.Context, err error) {
	status, code, message := mapError(err)
	WriteError(c, status, code, message)
}

func mapError(err error) (status int, code, message string) {
	if validationErr := validationError(err); validationErr != nil {
		return http.StatusBadRequest, CodeInvalidRequest, validationErr.Error()
	}

	switch {
	case errors.Is(err, errs.ErrUnauthorized),
		errors.Is(err, errs.ErrInvalidCredentials),
		errors.Is(err, errs.ErrInvalidRefreshToken):
		return http.StatusUnauthorized, CodeUnauthorized, errs.ErrUnauthorized.Error()

	case errors.Is(err, errs.ErrForbidden):
		return http.StatusForbidden, CodeForbidden, errs.ErrForbidden.Error()

	case errors.Is(err, errs.ErrRoomNotFound):
		return http.StatusNotFound, CodeRoomNotFound, errs.ErrRoomNotFound.Error()

	case errors.Is(err, errs.ErrSlotNotFound):
		return http.StatusNotFound, CodeSlotNotFound, errs.ErrSlotNotFound.Error()

	case errors.Is(err, errs.ErrBookingNotFound):
		return http.StatusNotFound, CodeBookingNotFound, errs.ErrBookingNotFound.Error()

	case errors.Is(err, errs.ErrUserNotFound),
		errors.Is(err, errs.ErrScheduleNotFound),
		errors.Is(err, errs.ErrScheduleRuleNotFound):
		return http.StatusNotFound, CodeNotFound, "resource not found"

	case errors.Is(err, errs.ErrSlotAlreadyBooked):
		return http.StatusConflict, CodeSlotAlreadyBooked, errs.ErrSlotAlreadyBooked.Error()

	case errors.Is(err, errs.ErrScheduleAlreadyExists):
		return http.StatusConflict, CodeScheduleExists, errs.ErrScheduleAlreadyExists.Error()

	default:
		return http.StatusInternalServerError, CodeInternalError, "internal server error"
	}
}

func validationError(err error) error {
	validationErrors := []error{
		errs.ErrUserEmailAlreadyExists,
		errs.ErrRoomNameAlreadyExists,
		errs.ErrSlotOverlap,
		errs.ErrSlotInPast,
		errs.ErrUserEmailRequired,
		errs.ErrUserRoleInvalid,
		errs.ErrPasswordRequired,
		errs.ErrPasswordTooLong,
		errs.ErrSlotRoomIDRequired,
		errs.ErrSlotStartAtRequired,
		errs.ErrSlotEndAtRequired,
		errs.ErrSlotTimeRangeInvalid,
		errs.ErrSlotDurationInvalid,
		errs.ErrRoomNameRequired,
		errs.ErrRoomCapacityInvalid,
		errs.ErrBookingSlotIDRequired,
		errs.ErrBookingUserIDRequired,
		errs.ErrBookingStatusInvalid,
		errs.ErrPaginationPageInvalid,
		errs.ErrPaginationPageSizeInvalid,
		errs.ErrScheduleRuleDayInvalid,
		errs.ErrScheduleRuleStartTimeInvalid,
		errs.ErrScheduleRuleEndTimeInvalid,
		errs.ErrScheduleRuleTimeRangeInvalid,
		errs.ErrScheduleRoomIDRequired,
		errs.ErrScheduleRulesRequired,
		errs.ErrScheduleDuplicateDay,
	}

	for _, validationErr := range validationErrors {
		if errors.Is(err, validationErr) {
			return validationErr
		}
	}

	return nil
}
