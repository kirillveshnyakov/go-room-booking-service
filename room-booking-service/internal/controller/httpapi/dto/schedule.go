package dto

import "time"

type CreateScheduleURI struct {
	RoomID string `uri:"roomId" binding:"required,uuid"`
}

type CreateScheduleRequest struct {
	Rules []ScheduleRuleRequest `json:"rules" binding:"required,min=1,max=7,dive"`
}

type ScheduleRuleRequest struct {
	DayOfWeek int    `json:"dayOfWeek" binding:"required,gte=1,lte=7"`
	StartTime string `json:"startTime" binding:"required,datetime=15:04"`
	EndTime   string `json:"endTime" binding:"required,datetime=15:04"`
}

type CreateScheduleResponse struct {
	Schedule ScheduleResponse `json:"schedule"`
}

type ScheduleResponse struct {
	ID        string                 `json:"id"`
	RoomID    string                 `json:"roomId"`
	Rules     []ScheduleRuleResponse `json:"rules"`
	CreatedAt time.Time              `json:"createdAt"`
}

type ScheduleRuleResponse struct {
	DayOfWeek int    `json:"dayOfWeek"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}
