package room

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/entity"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/errs"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres/transactor"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/port"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/repository/postgres/sqlcgen"
)

const (
	roomsNameUniqueConstraint       = "rooms_name_unique"
	roomsCapacityPositiveConstraint = "rooms_capacity_positive"
)

type roomRepository struct {
	queries *sqlcgen.Queries
}

func NewRoomRepository(db sqlcgen.DBTX) *roomRepository {
	return &roomRepository{
		queries: sqlcgen.New(db),
	}
}

func (repo *roomRepository) getQueries(ctx context.Context) *sqlcgen.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return repo.queries.WithTx(tx)
	}

	return repo.queries
}

func (repo *roomRepository) Create(
	ctx context.Context,
	params port.CreateRoomParams,
) (entity.Room, error) {
	room, err := repo.getQueries(ctx).CreateRoom(ctx, sqlcgen.CreateRoomParams{
		Name:        params.Name,
		Description: params.Description,
		Capacity:    int32(params.Capacity),
	})
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation &&
				pgErr.ConstraintName == roomsNameUniqueConstraint {
				return entity.Room{}, errs.ErrRoomNameAlreadyExists
			}
			if pgErr.Code == pgerrcode.CheckViolation &&
				pgErr.ConstraintName == roomsCapacityPositiveConstraint {
				return entity.Room{}, errs.ErrRoomCapacityInvalid
			}
		}

		return entity.Room{}, fmt.Errorf("room repository - create: %w", err)
	}

	return toEntityRoom(room), nil
}

func (repo *roomRepository) GetByID(
	ctx context.Context,
	roomID uuid.UUID,
) (entity.Room, error) {
	room, err := repo.getQueries(ctx).GetRoomByID(ctx, roomID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Room{}, errs.ErrRoomNotFound
		}

		return entity.Room{}, fmt.Errorf("room repository - get by id: %w", err)
	}

	return toEntityRoom(room), nil
}

func (repo *roomRepository) List(
	ctx context.Context,
) ([]entity.Room, error) {
	list, err := repo.getQueries(ctx).ListRooms(ctx)
	if err != nil {
		return nil, fmt.Errorf("room repository - list: %w", err)
	}

	rooms := make([]entity.Room, 0, len(list))
	for _, room := range list {
		rooms = append(rooms, toEntityRoom(room))
	}

	return rooms, nil
}
