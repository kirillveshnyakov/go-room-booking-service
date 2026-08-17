package slot

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
	slotsRoomIDFKConstraint      = "slots_room_id_fk"
	slotsDurationValidConstraint = "slots_duration_valid"
	slotsNoOverlapConstraint     = "slots_no_overlap"
)

type slotRepository struct {
	queries *sqlcgen.Queries
}

func NewSlotRepository(
	db sqlcgen.DBTX,
) *slotRepository {
	return &slotRepository{
		queries: sqlcgen.New(db),
	}
}

func (repo *slotRepository) getQueries(ctx context.Context) *sqlcgen.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return repo.queries.WithTx(tx)
	}

	return repo.queries
}

func (repo *slotRepository) Create(
	ctx context.Context,
	roomID uuid.UUID,
	startAts []time.Time,
) error {
	_, err := repo.getQueries(ctx).CreateSlotsForDate(ctx, sqlcgen.CreateSlotsForDateParams{
		RoomID:   roomID,
		StartAts: timeSliceToPg(startAts),
	})
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			switch {
			case pgErr.Code == pgerrcode.ForeignKeyViolation &&
				pgErr.ConstraintName == slotsRoomIDFKConstraint:
				return errs.ErrRoomNotFound
			case pgErr.Code == pgerrcode.CheckViolation &&
				pgErr.ConstraintName == slotsDurationValidConstraint:
				return errs.ErrSlotDurationInvalid
			case pgErr.Code == pgerrcode.ExclusionViolation &&
				pgErr.ConstraintName == slotsNoOverlapConstraint:
				return errs.ErrSlotOverlap
			}
		}

		return fmt.Errorf("slot repository - create: %w", err)
	}

	return nil
}

func (repo *slotRepository) GetByID(
	ctx context.Context,
	slotID uuid.UUID,
) (entity.Slot, error) {
	slot, err := repo.getQueries(ctx).GetSlotByID(ctx, slotID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Slot{}, errs.ErrSlotNotFound
		}
		return entity.Slot{}, fmt.Errorf("slot repository - get by id: %w", err)
	}

	return entity.Slot{
		ID:      slot.ID,
		RoomID:  slot.RoomID,
		StartAt: slot.StartAt.Time.UTC(),
		EndAt:   slot.EndAt.Time.UTC(),
	}, nil
}

func (repo *slotRepository) ListFree(
	ctx context.Context,
	roomID uuid.UUID,
	targetDate time.Time,
) ([]entity.Slot, error) {
	slots, err := repo.getQueries(ctx).ListFreeSlots(ctx, sqlcgen.ListFreeSlotsParams{
		RoomID:     roomID,
		TargetDate: dateToPg(targetDate),
	})
	if err != nil {
		return nil, fmt.Errorf("slot repository - list free: %w", err)
	}

	list := make([]entity.Slot, 0, len(slots))

	for _, slot := range slots {
		list = append(list, entity.Slot{
			ID:      slot.ID,
			RoomID:  slot.RoomID,
			StartAt: slot.StartAt.Time.UTC(),
			EndAt:   slot.EndAt.Time.UTC(),
		})
	}

	return list, nil
}

func dateToPg(value time.Time) pgtype.Date {
	date := value.UTC()

	return pgtype.Date{
		Time:  time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
}

func timeSliceToPg(values []time.Time) []pgtype.Timestamptz {
	result := make([]pgtype.Timestamptz, 0, len(values))
	for _, value := range values {
		result = append(result, pgtype.Timestamptz{
			Time:  value.UTC(),
			Valid: true,
		})
	}
	return result
}
