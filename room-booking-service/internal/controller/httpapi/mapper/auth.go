package mapper

import (
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/controller/httpapi/dto"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
)

func UserToResponse(user entity.User) dto.UserResponse {
	return dto.UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Role:      string(user.Role),
		CreatedAt: optionalUTCTime(user.CreatedAt),
	}
}

func AuthTokensToAccessTokenResponse(tokens port.AuthTokens) dto.AccessTokenResponse {
	return dto.AccessTokenResponse{
		AccessToken: tokens.AccessToken,
	}
}
