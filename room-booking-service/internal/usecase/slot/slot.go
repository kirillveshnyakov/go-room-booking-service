package slot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"go.uber.org/zap"
)

type (
	slotRepository interface {
		Create(ctx context.Context, roomID uuid.UUID, startAts []time.Time) error
		ListFree(ctx context.Context, roomID uuid.UUID, targetDate time.Time) ([]entity.Slot, error)
		CheckExistsForDate(ctx context.Context, roomID uuid.UUID, targetDate time.Time) (bool, error)
	}

	scheduleRepository interface {
		GetRuleForDay(ctx context.Context, roomID uuid.UUID, dayOfWeek entity.DayOfWeek) (entity.ScheduleRule, error)
	}

	roomRepository interface {
		GetByID(ctx context.Context, roomID uuid.UUID) (entity.Room, error)
	}
)

type slotService struct {
	slotRepository     slotRepository
	scheduleRepository scheduleRepository
	roomRepository     roomRepository
	logger             *zap.Logger
}

func NewSlotService(
	slotRepository slotRepository,
	scheduleRepository scheduleRepository,
	roomRepository roomRepository,
	logger *zap.Logger,
) *slotService {
	return &slotService{
		slotRepository:     slotRepository,
		scheduleRepository: scheduleRepository,
		roomRepository:     roomRepository,
		logger:             logger.Named("slot_usecase"),
	}
}

func (service *slotService) ListFree(
	ctx context.Context,
	roomID uuid.UUID,
	targetDate time.Time,
) ([]entity.Slot, error) {
	targetDate = targetDate.UTC()

	_, err := service.roomRepository.GetByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, errs.ErrRoomNotFound) {
			return nil, err
		}

		service.logger.Error(
			"room lookup failed",
			zap.Error(err),
		)

		return nil, fmt.Errorf("slot usecase - list free: %w", err)
	}

	ok, checkErr := service.slotRepository.CheckExistsForDate(ctx, roomID, targetDate)
	if checkErr != nil {
		service.logger.Error(
			"slot existence check failed",
			zap.Error(checkErr),
		)

		return nil, fmt.Errorf("slot usecase - list free: %w", checkErr)
	}

	if !ok {
		rule, ruleErr := service.scheduleRepository.GetRuleForDay(ctx, roomID, entity.GetDayOfWeek(targetDate))
		if ruleErr != nil {
			if errors.Is(ruleErr, errs.ErrScheduleRuleNotFound) ||
				errors.Is(ruleErr, errs.ErrScheduleNotFound) {
				return []entity.Slot{}, nil
			}

			service.logger.Error(
				"schedule rule lookup failed",
				zap.Error(ruleErr),
			)

			return nil, fmt.Errorf("slot usecase - list free: %w", ruleErr)
		}

		if err = service.generate(ctx, roomID, targetDate, rule); err != nil {
			return nil, fmt.Errorf("slot usecase - list free: %w", err)
		}
	}

	slots, listErr := service.slotRepository.ListFree(ctx, roomID, targetDate)
	if listErr != nil {
		service.logger.Error(
			"slot list failed",
			zap.Error(listErr),
		)

		return nil, fmt.Errorf("slot usecase - list free: %w", listErr)
	}

	return slots, nil
}

const slotDuration = 30 * time.Minute

func (service *slotService) generate(
	ctx context.Context,
	roomID uuid.UUID,
	targetDate time.Time,
	rule entity.ScheduleRule,
) error {
	dayStart := time.Date(
		targetDate.Year(),
		targetDate.Month(),
		targetDate.Day(),
		0, 0, 0, 0,
		time.UTC,
	)

	start := dayStart.Add(rule.StartTime)
	ruleEnd := dayStart.Add(rule.EndTime)

	startAts := make([]time.Time, 0)

	for {
		end := start.Add(slotDuration)

		if end.After(ruleEnd) {
			break
		}

		startAts = append(startAts, start)
		start = end
	}

	if len(startAts) == 0 {
		return nil
	}

	if err := service.slotRepository.Create(ctx, roomID, startAts); err != nil {
		service.logger.Error(
			"slot generation failed",
			zap.Error(err),
		)

		return fmt.Errorf("slot usecase - generate: %w", err)
	}

	service.logger.Info(
		"slots generated",
		zap.String("room_id", roomID.String()),
		zap.Time("date", targetDate),
		zap.Int("slots_count", len(startAts)),
	)

	return nil
}
