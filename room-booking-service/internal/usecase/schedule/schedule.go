package schedule

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
	"go.uber.org/zap"
)

type (
	scheduleRepository interface {
		Create(ctx context.Context, schedule entity.Schedule) (entity.Schedule, error)
	}
)

type scheduleService struct {
	scheduleRepository scheduleRepository
	logger             *zap.Logger
}

func NewScheduleService(
	scheduleRepository scheduleRepository,
	logger *zap.Logger,
) *scheduleService {
	return &scheduleService{
		scheduleRepository: scheduleRepository,
		logger:             logger,
	}
}

func (service *scheduleService) Create(
	ctx context.Context,
	actor entity.Identity,
	roomID uuid.UUID,
	rules []entity.ScheduleRule,
) (entity.Schedule, error) {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("schedule_usecase")
	if err := actor.Validate(); err != nil || actor.Role != entity.UserRoleAdmin {
		return entity.Schedule{}, errs.ErrForbidden
	}

	schedule := entity.Schedule{
		RoomID: roomID,
		Rules:  rules,
	}
	if err := schedule.Validate(); err != nil {
		return entity.Schedule{}, fmt.Errorf("schedule usecase - create: validation error: %w", err)
	}

	createdSchedule, err := service.scheduleRepository.Create(ctx, schedule)
	if err != nil {
		if errors.Is(err, errs.ErrScheduleAlreadyExists) ||
			errors.Is(err, errs.ErrRoomNotFound) {
			return entity.Schedule{}, err
		}

		log.Error(
			"schedule creation failed",
			zap.Error(err),
		)

		return entity.Schedule{}, fmt.Errorf("schedule usecase - create: %w", err)
	}

	log.Info(
		"schedule created",
		zap.String("schedule_id", createdSchedule.ID.String()),
		zap.String("room_id", createdSchedule.RoomID.String()),
		zap.Int("rules_count", len(createdSchedule.Rules)),
	)

	return createdSchedule, nil
}
