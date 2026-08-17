package schedule

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
	scheduleRoomIDUnique       = "schedules_room_id_unique"
	scheduleRoomIDFKConstraint = "schedules_room_id_fk"

	scheduleRulesPrimaryKeyConstraint = "schedule_rules_unique_day"
	scheduleRulesScheduleFKConstraint = "schedule_rules_schedule_id_fk"
	scheduleRulesDayValidConstraint   = "schedule_rules_day_valid"
	scheduleRulesTimeValidConstraint  = "schedule_rules_time_valid"
)

type scheduleRepository struct {
	queries    *sqlcgen.Queries
	transactor transactor.Transactor
}

func NewScheduleRepository(
	db sqlcgen.DBTX,
	transactor transactor.Transactor,
) *scheduleRepository {
	return &scheduleRepository{
		queries:    sqlcgen.New(db),
		transactor: transactor,
	}
}

func (repo *scheduleRepository) getQueries(ctx context.Context) *sqlcgen.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return repo.queries.WithTx(tx)
	}

	return repo.queries
}

func (repo *scheduleRepository) Create(
	ctx context.Context,
	schedule entity.Schedule,
) (entity.Schedule, error) {
	err := repo.transactor.WithTx(ctx, func(ctx context.Context) error {
		createdSchedule, err := repo.getQueries(ctx).CreateSchedule(ctx, schedule.RoomID)
		if err != nil {
			var pgErr *pgconn.PgError

			if errors.As(err, &pgErr) {
				switch {
				case pgErr.Code == pgerrcode.UniqueViolation &&
					pgErr.ConstraintName == scheduleRoomIDUnique:
					return errs.ErrScheduleAlreadyExists
				case pgErr.Code == pgerrcode.ForeignKeyViolation &&
					pgErr.ConstraintName == scheduleRoomIDFKConstraint:
					return errs.ErrRoomNotFound
				}
			}

			return fmt.Errorf("schedule repository - create: %w", err)
		}

		schedule.ID = createdSchedule.ID
		schedule.CreatedAt = createdSchedule.CreatedAt.Time.UTC()

		for _, rule := range schedule.Rules {
			err = repo.getQueries(ctx).CreateScheduleRule(ctx, sqlcgen.CreateScheduleRuleParams{
				ScheduleID: createdSchedule.ID,
				DayOfWeek:  int32(rule.DayOfWeek),
				StartAt:    timeToPg(rule.StartTime),
				EndAt:      timeToPg(rule.EndTime),
			})
			if err != nil {
				var pgErr *pgconn.PgError

				if errors.As(err, &pgErr) {
					switch {
					case pgErr.Code == pgerrcode.UniqueViolation &&
						pgErr.ConstraintName == scheduleRulesPrimaryKeyConstraint:
						return errs.ErrScheduleDuplicateDay
					case pgErr.Code == pgerrcode.ForeignKeyViolation &&
						pgErr.ConstraintName == scheduleRulesScheduleFKConstraint:
						return errs.ErrScheduleNotFound
					case pgErr.Code == pgerrcode.CheckViolation &&
						pgErr.ConstraintName == scheduleRulesDayValidConstraint:
						return errs.ErrScheduleRuleDayInvalid
					case pgErr.Code == pgerrcode.CheckViolation &&
						pgErr.ConstraintName == scheduleRulesTimeValidConstraint:
						return errs.ErrScheduleRuleTimeRangeInvalid
					}
				}

				return fmt.Errorf("schedule repository - create rule: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return entity.Schedule{}, err
	}

	return schedule, nil
}

func (repo *scheduleRepository) GetRuleForDay(
	ctx context.Context,
	roomID uuid.UUID,
	dayOfWeek entity.DayOfWeek,
) (entity.ScheduleRule, error) {
	rule, err := repo.getQueries(ctx).GetScheduleRuleForDay(ctx, sqlcgen.GetScheduleRuleForDayParams{
		RoomID:    roomID,
		DayOfWeek: int32(dayOfWeek),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.ScheduleRule{}, errs.ErrScheduleNotFound
		}
		return entity.ScheduleRule{}, fmt.Errorf("schedule repository - get rule for day: %w", err)
	}

	if !rule.DayOfWeek.Valid {
		return entity.ScheduleRule{}, errs.ErrScheduleRuleNotFound
	}

	return entity.ScheduleRule{
		DayOfWeek: entity.DayOfWeek(rule.DayOfWeek.Int32),
		StartTime: time.Duration(rule.StartAt.Microseconds) * time.Microsecond,
		EndTime:   time.Duration(rule.EndAt.Microseconds) * time.Microsecond,
	}, nil
}

func timeToPg(t time.Duration) pgtype.Time {
	return pgtype.Time{
		Microseconds: t.Microseconds(),
		Valid:        true,
	}
}
