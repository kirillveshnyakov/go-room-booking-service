package entity

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

func (r UserRole) IsValid() bool {
	switch r {
	case UserRoleAdmin, UserRoleUser:
		return true
	default:
		return false
	}
}

type DayOfWeek uint8

const (
	DayOfWeekMonday DayOfWeek = iota + 1
	DayOfWeekTuesday
	DayOfWeekWednesday
	DayOfWeekThursday
	DayOfWeekFriday
	DayOfWeekSaturday
	DayOfWeekSunday
)

func (d DayOfWeek) IsValid() bool {
	switch d {
	case DayOfWeekMonday, DayOfWeekTuesday, DayOfWeekWednesday, DayOfWeekThursday,
		DayOfWeekFriday, DayOfWeekSaturday, DayOfWeekSunday:
		return true
	default:
		return false
	}
}

type BookingStatus string

const (
	BookingStatusActive    BookingStatus = "active"
	BookingStatusCancelled BookingStatus = "cancelled"
)

func (s BookingStatus) IsValid() bool {
	switch s {
	case BookingStatusActive, BookingStatusCancelled:
		return true
	default:
		return false
	}
}
