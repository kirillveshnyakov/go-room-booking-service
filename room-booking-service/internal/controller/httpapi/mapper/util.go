package mapper

import (
	"fmt"
	"time"
)

const (
	timeLayout = "15:04"
)

func optionalString(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func optionalUTCTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	value = value.UTC()

	return &value
}

func timeStringToDuration(value string) (time.Duration, error) {
	parsedTime, err := time.Parse(timeLayout, value)
	if err != nil {
		return 0, err
	}

	return time.Duration(parsedTime.Hour())*time.Hour +
		time.Duration(parsedTime.Minute())*time.Minute, nil
}

func durationToTimeString(value time.Duration) string {
	totalMinutes := int(value / time.Minute)

	return fmt.Sprintf("%02d:%02d", totalMinutes/60, totalMinutes%60)
}
