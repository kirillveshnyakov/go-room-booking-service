package entity

import (
	"strings"
	"time"
)

func normalizeText(value string) string {
	return strings.TrimSpace(value)
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}

	return value.UTC()
}

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

func (r UserRole) Normalize() UserRole {
	return UserRole(strings.ToLower(normalizeText(string(r))))
}

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

func GetDayOfWeek(date time.Time) DayOfWeek {
	weekday := date.Weekday()

	if weekday == time.Sunday {
		return DayOfWeekSunday
	}

	return DayOfWeek(weekday)
}

type BookingStatus string

const (
	BookingStatusActive    BookingStatus = "active"
	BookingStatusCancelled BookingStatus = "cancelled"
)

func (s BookingStatus) Normalize() BookingStatus {
	return BookingStatus(strings.ToLower(normalizeText(string(s))))
}

func (s BookingStatus) IsValid() bool {
	switch s {
	case BookingStatusActive, BookingStatusCancelled:
		return true
	default:
		return false
	}
}
