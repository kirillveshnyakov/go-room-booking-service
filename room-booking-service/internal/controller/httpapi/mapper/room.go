package mapper

import (
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
)

func CreateRoomRequestToParams(request dto.CreateRoomRequest) port.CreateRoomParams {
	return port.CreateRoomParams{
		Name:        request.Name,
		Description: request.Description,
		Capacity:    request.Capacity,
	}
}

func RoomToResponse(room entity.Room) dto.RoomResponse {
	return dto.RoomResponse{
		ID:          room.ID.String(),
		Name:        room.Name,
		Description: room.Description,
		Capacity:    room.Capacity,
		CreatedAt:   optionalUTCTime(room.CreatedAt),
	}
}

func RoomsToResponse(rooms []entity.Room) []dto.RoomResponse {
	result := make([]dto.RoomResponse, 0, len(rooms))
	for _, room := range rooms {
		result = append(result, RoomToResponse(room))
	}

	return result
}
