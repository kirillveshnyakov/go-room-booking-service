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
	"go.uber.org/zap"
)

type (
	bookingRepository interface {
		Create(ctx context.Context, slotID uuid.UUID, userID uuid.UUID, conferenceLink string) (entity.Booking, error)
		Cancel(ctx context.Context, bookingID uuid.UUID, userID uuid.UUID) (entity.Booking, error)
		List(ctx context.Context, pageLimit int, pageOffset int) ([]entity.Booking, error)
		ListUserFuture(ctx context.Context, userID uuid.UUID) ([]entity.Booking, error)
		Count(ctx context.Context) (int64, error)
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
		logger:                  logger.Named("booking_usecase"),
	}
}

func (service *bookingService) Create(
	ctx context.Context,
	params port.CreateBookingParams,
) (entity.Booking, error) {
	var conferenceLink string

	if params.CreateConferenceLink {
		link, err := service.conferenceLinkGenerator.Generate(ctx)
		if err != nil {
			service.logger.Error(
				"booking creation failed",
				zap.Error(err),
			)

			return entity.Booking{}, fmt.Errorf("booking usecase - create: generate conference link: %w", err)
		}

		conferenceLink = link
	}

	booking, err := service.bookingRepository.Create(ctx, params.SlotID, params.UserID, conferenceLink)
	if err != nil {
		if errors.Is(err, errs.ErrSlotNotFound) ||
			errors.Is(err, errs.ErrSlotAlreadyBooked) ||
			errors.Is(err, errs.ErrSlotInPast) ||
			errors.Is(err, errs.ErrUserNotFound) {
			return entity.Booking{}, err
		}

		service.logger.Error(
			"booking creation failed",
			zap.Error(err),
		)

		return entity.Booking{}, fmt.Errorf("booking usecase - create: %w", err)
	}

	service.logger.Info(
		"booking created",
		zap.String("booking_id", booking.ID.String()),
		zap.String("slot_id", booking.SlotID.String()),
		zap.String("user_id", booking.UserID.String()),
	)

	return booking, nil
}

func (service *bookingService) List(
	ctx context.Context,
	page int,
	pageSize int,
) ([]entity.Booking, int64, error) {
	if pageSize < 1 || pageSize > maxPageSize {
		return nil, 0, fmt.Errorf("booking usecase - list: validation error: %w", errs.ErrPaginationPageSizeInvalid)
	}
	if page < 1 || page > math.MaxInt/pageSize+1 {
		return nil, 0, fmt.Errorf("booking usecase - list: validation error: %w", errs.ErrPaginationPageInvalid)
	}

	pageOffset := (page - 1) * pageSize

	list, err := service.bookingRepository.List(ctx, pageSize, pageOffset)
	if err != nil {
		service.logger.Error(
			"booking list failed",
			zap.Error(err),
		)

		return nil, 0, fmt.Errorf("booking usecase - list: %w", err)
	}

	total, countErr := service.bookingRepository.Count(ctx)
	if countErr != nil {
		service.logger.Error(
			"booking count failed",
			zap.Error(countErr),
		)

		return nil, 0, fmt.Errorf("booking usecase - list: %w", countErr)
	}

	return list, total, nil
}

func (service *bookingService) ListUserFuture(
	ctx context.Context,
	userID uuid.UUID,
) ([]entity.Booking, error) {
	list, err := service.bookingRepository.ListUserFuture(ctx, userID)
	if err != nil {
		service.logger.Error(
			"user bookings list failed",
			zap.Error(err),
		)

		return nil, fmt.Errorf("booking usecase - list user future: %w", err)
	}

	return list, nil
}

func (service *bookingService) Cancel(
	ctx context.Context,
	userID uuid.UUID,
	bookingID uuid.UUID,
) (entity.Booking, error) {
	canceledBooking, err := service.bookingRepository.Cancel(ctx, bookingID, userID)
	if err != nil {
		if errors.Is(err, errs.ErrBookingNotFound) ||
			errors.Is(err, errs.ErrForbidden) {
			return entity.Booking{}, err
		}

		service.logger.Error(
			"booking cancel failed",
			zap.Error(err),
		)

		return entity.Booking{}, fmt.Errorf("booking usecase - cancel: %w", err)
	}

	service.logger.Info(
		"booking cancelled",
		zap.String("booking_id", canceledBooking.ID.String()),
		zap.String("user_id", userID.String()),
	)

	return canceledBooking, nil
}
