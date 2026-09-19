package booking

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/requestctx"
	"go.uber.org/zap"
)

type (
	bookingRepository interface {
		Create(ctx context.Context, slotID uuid.UUID, userID uuid.UUID, conferenceLink string) (entity.Booking, error)
		GetByID(ctx context.Context, bookingID uuid.UUID) (entity.Booking, error)
		Cancel(ctx context.Context, bookingID uuid.UUID) error
		List(ctx context.Context, pageLimit int, pageOffset int) ([]entity.Booking, int64, error)
		ListUserFuture(ctx context.Context, userID uuid.UUID) ([]entity.Booking, error)
	}

	conferenceLinkGenerator interface {
		Generate(ctx context.Context) (string, error)
	}
)

type bookingService struct {
	bookingRepository       bookingRepository
	conferenceLinkGenerator conferenceLinkGenerator
	logger                  *zap.Logger
}

const maxPageSize = 100

func NewBookingService(
	bookingRepository bookingRepository,
	conferenceLinkGenerator conferenceLinkGenerator,
	logger *zap.Logger,
) *bookingService {
	return &bookingService{
		bookingRepository:       bookingRepository,
		conferenceLinkGenerator: conferenceLinkGenerator,
		logger:                  logger,
	}
}

func (service *bookingService) Create(
	ctx context.Context,
	actor entity.Identity,
	params port.CreateBookingParams,
) (entity.Booking, error) {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("booking_usecase")
	if err := actor.Validate(); err != nil || actor.Role != entity.UserRoleUser {
		return entity.Booking{}, errs.ErrForbidden
	}

	var conferenceLink string

	if params.CreateConferenceLink {
		link, err := service.conferenceLinkGenerator.Generate(ctx)
		if err != nil {
			log.Error(
				"booking creation failed",
				zap.Error(err),
			)

			return entity.Booking{}, fmt.Errorf("booking usecase - create: generate conference link: %w", err)
		}

		conferenceLink = link
	}

	booking, err := service.bookingRepository.Create(ctx, params.SlotID, actor.UserID, conferenceLink)
	if err != nil {
		if errors.Is(err, errs.ErrSlotNotFound) ||
			errors.Is(err, errs.ErrSlotAlreadyBooked) ||
			errors.Is(err, errs.ErrSlotInPast) ||
			errors.Is(err, errs.ErrUserNotFound) {
			return entity.Booking{}, err
		}

		log.Error(
			"booking creation failed",
			zap.Error(err),
		)

		return entity.Booking{}, fmt.Errorf("booking usecase - create: %w", err)
	}

	log.Info(
		"booking created",
		zap.String("booking_id", booking.ID.String()),
		zap.String("slot_id", booking.SlotID.String()),
		zap.String("user_id", booking.UserID.String()),
	)

	return booking, nil
}

func (service *bookingService) List(
	ctx context.Context,
	actor entity.Identity,
	page int,
	pageSize int,
) ([]entity.Booking, int64, error) {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("booking_usecase")
	if err := actor.Validate(); err != nil || actor.Role != entity.UserRoleAdmin {
		return nil, 0, errs.ErrForbidden
	}

	if pageSize < 1 || pageSize > maxPageSize {
		return nil, 0, fmt.Errorf("booking usecase - list: validation error: %w", errs.ErrPaginationPageSizeInvalid)
	}
	if page < 1 || page > math.MaxInt/pageSize+1 {
		return nil, 0, fmt.Errorf("booking usecase - list: validation error: %w", errs.ErrPaginationPageInvalid)
	}

	pageOffset := (page - 1) * pageSize

	list, total, err := service.bookingRepository.List(ctx, pageSize, pageOffset)
	if err != nil {
		log.Error(
			"booking list failed",
			zap.Error(err),
		)

		return nil, 0, fmt.Errorf("booking usecase - list: %w", err)
	}

	return list, total, nil
}

func (service *bookingService) ListMy(
	ctx context.Context,
	actor entity.Identity,
) ([]entity.Booking, error) {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("booking_usecase")
	if err := actor.Validate(); err != nil || actor.Role != entity.UserRoleUser {
		return nil, errs.ErrForbidden
	}

	list, err := service.bookingRepository.ListUserFuture(ctx, actor.UserID)
	if err != nil {
		log.Error(
			"user bookings list failed",
			zap.Error(err),
		)

		return nil, fmt.Errorf("booking usecase - list user future: %w", err)
	}

	return list, nil
}

func (service *bookingService) Cancel(
	ctx context.Context,
	actor entity.Identity,
	bookingID uuid.UUID,
) (entity.Booking, error) {
	log := requestctx.LoggerOrDefault(ctx, service.logger).Named("booking_usecase")
	if err := actor.Validate(); err != nil {
		return entity.Booking{}, errs.ErrForbidden
	}

	booking, err := service.bookingRepository.GetByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, errs.ErrBookingNotFound) {
			return entity.Booking{}, err
		}

		log.Error(
			"booking lookup for cancellation failed",
			zap.Error(err),
		)

		return entity.Booking{}, fmt.Errorf("booking usecase - cancel: %w", err)
	}
	if actor.Role == entity.UserRoleUser && booking.UserID != actor.UserID {
		return entity.Booking{}, errs.ErrForbidden
	}

	err = service.bookingRepository.Cancel(ctx, bookingID)
	if err != nil {
		if errors.Is(err, errs.ErrBookingNotFound) {
			return entity.Booking{}, err
		}

		log.Error(
			"booking cancel failed",
			zap.Error(err),
		)

		return entity.Booking{}, fmt.Errorf("booking usecase - cancel: %w", err)
	}
	booking.Status = entity.BookingStatusCancelled

	log.Info(
		"booking cancelled",
		zap.String("booking_id", booking.ID.String()),
		zap.String("user_id", actor.UserID.String()),
	)

	return booking, nil
}
