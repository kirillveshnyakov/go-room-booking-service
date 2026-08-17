package booking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres/transactor"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/sqlcgen"
)

const (
	bookingsSlotIDFKConstraint    = "bookings_slot_id_fk"
	bookingsUserIDFKConstraint    = "bookings_user_id_fk"
	bookingsActiveSlotUniqueIndex = "bookings_one_active_per_slot"
)

type bookingRepository struct {
	queries *sqlcgen.Queries
}

func NewBookingRepository(db sqlcgen.DBTX) *bookingRepository {
	return &bookingRepository{
		queries: sqlcgen.New(db),
	}
}

func (repo *bookingRepository) getQueries(ctx context.Context) *sqlcgen.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return repo.queries.WithTx(tx)
	}

	return repo.queries
}

func (repo *bookingRepository) Create(
	ctx context.Context,
	slotID uuid.UUID,
	userID uuid.UUID,
	conferenceLink string,
) (entity.Booking, error) {
	booking, err := repo.getQueries(ctx).CreateBooking(ctx, sqlcgen.CreateBookingParams{
		UserID:         userID,
		ConferenceLink: textToPg(conferenceLink),
		SlotID:         slotID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Booking{}, repo.resolveMissingBookingSlot(ctx, slotID)
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch {
			case pgErr.Code == pgerrcode.UniqueViolation &&
				pgErr.ConstraintName == bookingsActiveSlotUniqueIndex:
				return entity.Booking{}, errs.ErrSlotAlreadyBooked
			case pgErr.Code == pgerrcode.ForeignKeyViolation &&
				pgErr.ConstraintName == bookingsSlotIDFKConstraint:
				return entity.Booking{}, errs.ErrSlotNotFound
			case pgErr.Code == pgerrcode.ForeignKeyViolation &&
				pgErr.ConstraintName == bookingsUserIDFKConstraint:
				return entity.Booking{}, errs.ErrUserNotFound
			}
		}

		return entity.Booking{}, fmt.Errorf("booking repository - create: %w", err)
	}

	return bookingToEntity(booking), nil
}

func (repo *bookingRepository) GetByID(
	ctx context.Context,
	bookingID uuid.UUID,
) (entity.Booking, error) {
	booking, err := repo.getQueries(ctx).GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Booking{}, errs.ErrBookingNotFound
		}

		return entity.Booking{}, fmt.Errorf("booking repository - get by id: %w", err)
	}

	return bookingToEntity(booking), nil
}

func (repo *bookingRepository) Cancel(
	ctx context.Context,
	bookingID uuid.UUID,
	userID uuid.UUID,
) (entity.Booking, error) {
	booking, err := repo.GetByID(ctx, bookingID)
	if err != nil {
		return entity.Booking{}, err
	}
	if booking.UserID != userID {
		return entity.Booking{}, errs.ErrForbidden
	}

	count, err := repo.getQueries(ctx).CancelBooking(ctx, sqlcgen.CancelBookingParams{
		BookingID: bookingID,
		UserID:    userID,
	})
	if err != nil {
		return entity.Booking{}, fmt.Errorf("booking repository - cancel: %w", err)
	}
	if count == 0 {
		return entity.Booking{}, errs.ErrBookingNotFound
	}

	booking.Status = entity.BookingStatusCancelled
	return booking, nil
}

func (repo *bookingRepository) List(
	ctx context.Context,
	pageLimit int,
	pageOffset int,
) ([]entity.Booking, error) {
	bookings, err := repo.getQueries(ctx).ListBookings(ctx, sqlcgen.ListBookingsParams{
		PageLimit:  int32(pageLimit),
		PageOffset: int32(pageOffset),
	})
	if err != nil {
		return nil, fmt.Errorf("booking repository - list: %w", err)
	}

	result := make([]entity.Booking, 0, len(bookings))
	for _, booking := range bookings {
		result = append(result, bookingToEntity(booking))
	}

	return result, nil
}

func (repo *bookingRepository) ListUserFuture(
	ctx context.Context,
	userID uuid.UUID,
) ([]entity.Booking, error) {
	bookings, err := repo.getQueries(ctx).ListUserFutureBookings(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("booking repository - list user future: %w", err)
	}

	result := make([]entity.Booking, 0, len(bookings))
	for _, booking := range bookings {
		result = append(result, entity.Booking{
			ID:             booking.BookingID,
			SlotID:         booking.SlotID,
			UserID:         booking.UserID,
			Status:         entity.BookingStatus(booking.Status),
			ConferenceLink: textFromPg(booking.ConferenceLink),
			CreatedAt:      booking.CreatedAt.Time.UTC(),
		})
	}

	return result, nil
}

func (repo *bookingRepository) Count(ctx context.Context) (int64, error) {
	count, err := repo.getQueries(ctx).CountBookings(ctx)
	if err != nil {
		return 0, fmt.Errorf("booking repository - count: %w", err)
	}

	return count, nil
}

func (repo *bookingRepository) resolveMissingBookingSlot(
	ctx context.Context,
	slotID uuid.UUID,
) error {
	slot, err := repo.getQueries(ctx).GetSlotByID(ctx, slotID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errs.ErrSlotNotFound
		}

		return fmt.Errorf("booking repository - get slot for create: %w", err)
	}
	if !slot.StartAt.Time.After(time.Now()) {
		return errs.ErrSlotInPast
	}

	return fmt.Errorf("booking repository - create: future slot was not inserted")
}

func bookingToEntity(src sqlcgen.Booking) entity.Booking {
	return entity.Booking{
		ID:             src.ID,
		SlotID:         src.SlotID,
		UserID:         src.UserID,
		Status:         entity.BookingStatus(src.Status),
		ConferenceLink: textFromPg(src.ConferenceLink),
		CreatedAt:      src.CreatedAt.Time.UTC(),
	}
}

func textToPg(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}

	return pgtype.Text{
		String: value,
		Valid:  true,
	}
}

func textFromPg(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}

	return value.String
}
