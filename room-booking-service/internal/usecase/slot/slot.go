package slot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
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
		logger:             logger,
	}
}

func (service *slotService) ListFree(
	ctx context.Context,
	actor entity.Identity,
	roomID uuid.UUID,
	targetDate time.Time,
) ([]entity.Slot, error) {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("slot_usecase")
	if err := actor.Validate(); err != nil {
		return nil, errs.ErrForbidden
	}

	targetDate = targetDate.UTC()

	_, err := service.roomRepository.GetByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, errs.ErrRoomNotFound) {
			return nil, err
		}

		log.Error(
			"room lookup failed",
			zap.Error(err),
		)

		return nil, fmt.Errorf("slot usecase - list free: %w", err)
	}

	ok, checkErr := service.slotRepository.CheckExistsForDate(ctx, roomID, targetDate)
	if checkErr != nil {
		log.Error(
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

			log.Error(
				"schedule rule lookup failed",
				zap.Error(ruleErr),
			)

			return nil, fmt.Errorf("slot usecase - list free: %w", ruleErr)
		}

		if err = service.generate(ctx, log, roomID, targetDate, rule); err != nil {
			return nil, fmt.Errorf("slot usecase - list free: %w", err)
		}
	}

	slots, listErr := service.slotRepository.ListFree(ctx, roomID, targetDate)
	if listErr != nil {
		log.Error(
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
	log *zap.Logger,
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
		log.Error(
			"slot generation failed",
			zap.Error(err),
		)

		return fmt.Errorf("slot usecase - generate: %w", err)
	}

	log.Info(
		"slots generated",
		zap.String("room_id", roomID.String()),
		zap.Time("date", targetDate),
		zap.Int("slots_count", len(startAts)),
	)

	return nil
}
