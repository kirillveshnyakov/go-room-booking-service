package room

import (
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/sqlcgen"
)

func toEntityRoom(src sqlcgen.Room) entity.Room {
	return entity.Room{
		ID:          src.ID,
		Name:        src.Name,
		Description: src.Description,
		Capacity:    int(src.Capacity),
		CreatedAt:   src.CreatedAt.Time.UTC(),
	}
}
