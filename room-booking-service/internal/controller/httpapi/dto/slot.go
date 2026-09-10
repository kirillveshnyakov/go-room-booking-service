package dto

import "time"

type ListFreeSlotsURI struct {
	RoomID string `uri:"roomId" binding:"required,uuid"`
}

type ListFreeSlotsQuery struct {
	Date string `form:"date" binding:"required,datetime=2006-01-02"`
}

type ListFreeSlotsResponse struct {
	Slots []SlotResponse `json:"slots"`
}

type SlotResponse struct {
	ID      string    `json:"id"`
	RoomID  string    `json:"roomId"`
	StartAt time.Time `json:"start"`
	EndAt   time.Time `json:"end"`
}
