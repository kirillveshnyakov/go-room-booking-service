package dto

import "time"

type CreateRoomRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Capacity    int    `json:"capacity" binding:"required,gt=0"`
}

type CreateRoomResponse struct {
	Room RoomResponse `json:"room"`
}

type ListRoomsResponse struct {
	Rooms []RoomResponse `json:"rooms"`
}

type RoomResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Capacity    int        `json:"capacity"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
}
