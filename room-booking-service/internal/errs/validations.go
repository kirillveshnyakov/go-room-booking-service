package errs

import "errors"

var (
	ErrUserEmailRequired         = errors.New("user email is required")
	ErrUserRoleInvalid           = errors.New("user role is invalid")
	ErrPasswordRequired          = errors.New("password is required")
	ErrPasswordTooLong           = errors.New("password must not exceed 72 bytes")
	ErrIdentityUserIDRequired    = errors.New("identity user id is required")
	ErrIdentitySessionIDRequired = errors.New("identity session id is required")

	ErrSlotRoomIDRequired   = errors.New("slot room id is required")
	ErrSlotStartAtRequired  = errors.New("slot start time is required")
	ErrSlotEndAtRequired    = errors.New("slot end time is required")
	ErrSlotTimeRangeInvalid = errors.New("slot end time must be after start time")
	ErrSlotDurationInvalid  = errors.New("slot duration must be 30 minutes")

	ErrRoomNameRequired    = errors.New("room name is required")
	ErrRoomCapacityInvalid = errors.New("room capacity must be positive")

	ErrBookingSlotIDRequired     = errors.New("booking slot id is required")
	ErrBookingUserIDRequired     = errors.New("booking user id is required")
	ErrBookingStatusInvalid      = errors.New("booking status is invalid")
	ErrPaginationPageInvalid     = errors.New("page must be greater than zero")
	ErrPaginationPageSizeInvalid = errors.New("page size must be between 1 and 100")

	ErrScheduleRuleDayInvalid       = errors.New("schedule rule day is invalid")
	ErrScheduleRuleStartTimeInvalid = errors.New("schedule rule start time is invalid")
	ErrScheduleRuleEndTimeInvalid   = errors.New("schedule rule end time is invalid")
	ErrScheduleRuleTimeRangeInvalid = errors.New("schedule rule time range is invalid")

	ErrScheduleRoomIDRequired = errors.New("schedule room id is required")
	ErrScheduleRulesRequired  = errors.New("schedule rules are required")
	ErrScheduleDuplicateDay   = errors.New("duplicate schedule rule day")

	ErrSessionUserIDRequired      = errors.New("session user id is required")
	ErrSessionRefreshHashRequired = errors.New("session refresh hash is required")
	ErrSessionExpiresAtRequired   = errors.New("session expires at required")
	ErrSessionExpirationInvalid   = errors.New("session expiration invalid")
	ErrSessionRevokedAtInvalid    = errors.New("session revoked at required")
)
