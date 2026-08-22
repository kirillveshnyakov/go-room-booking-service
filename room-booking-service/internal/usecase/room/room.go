package room

import (
	"context"
	"errors"
	"fmt"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
	"go.uber.org/zap"
)

type (
	roomRepository interface {
		Create(ctx context.Context, room entity.Room) (entity.Room, error)
		List(ctx context.Context) ([]entity.Room, error)
	}
)

type roomService struct {
	roomRepository roomRepository
	logger         *zap.Logger
}

func NewRoomService(
	roomRepository roomRepository,
	logger *zap.Logger,
) *roomService {
	return &roomService{
		roomRepository: roomRepository,
		logger:         logger.Named("room_usecase"),
	}
}

func (service *roomService) Create(
	ctx context.Context,
	params port.CreateRoomParams,
) (entity.Room, error) {
	room := entity.Room{
		Name:        params.Name,
		Description: params.Description,
		Capacity:    params.Capacity,
	}
	if err := room.Validate(); err != nil {
		return entity.Room{}, fmt.Errorf("room usecase - create: validation error: %w", err)
	}

	createdRoom, err := service.roomRepository.Create(ctx, room)
	if err != nil {
		if errors.Is(err, errs.ErrRoomNameAlreadyExists) {
			return entity.Room{}, err
		}

		service.logger.Error(
			"room creation failed",
			zap.Error(err),
		)

		return entity.Room{}, fmt.Errorf("room usecase - create: %w", err)
	}

	service.logger.Info(
		"room created",
		zap.String("room_id", createdRoom.ID.String()),
	)

	return createdRoom, nil
}

func (service *roomService) List(ctx context.Context) ([]entity.Room, error) {
	list, err := service.roomRepository.List(ctx)
	if err != nil {
		service.logger.Error(
			"room list failed",
			zap.Error(err),
		)

		return nil, fmt.Errorf("room usecase - list: %w", err)
	}

	return list, nil
}
