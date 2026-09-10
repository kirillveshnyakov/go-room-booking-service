package mapper

import (
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
)

func UserToResponse(user entity.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Role:      string(user.Role),
		CreatedAt: optionalUTCTime(user.CreatedAt),
	}
}
