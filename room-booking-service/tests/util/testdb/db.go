package testdb

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres/migrate"
	"github.com/stretchr/testify/require"
)

const lockKey int64 = 0x524f4f4d424b5431
const fixtureTimeout = 10 * time.Second

type DB struct {
	Pool *pgxpool.Pool
	t    *testing.T
}

// New prepares the shared test database for one test. Every test using this
// database must call New and must not run in parallel: the advisory lock
// serializes participating test packages, and TRUNCATE clears previous fixtures
// at the start of each test. Fixtures are intentionally left after the last test.
func New(t *testing.T) *DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("integration test requires TEST_DATABASE_URL pointing to a dedicated *_test database")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(cfg.ConnConfig.Database, "_test"),
		"TEST_DATABASE_URL must point to a dedicated *_test database")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := postgres.NewPool(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	conn, err := pool.Acquire(ctx)
	require.NoError(t, err)
	t.Cleanup(conn.Release)
	_, err = conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey)
	require.NoError(t, err)
	t.Cleanup(func() {
		unlockCtx, unlockCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer unlockCancel()
		_, unlockErr := conn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", lockKey)
		require.NoError(t, unlockErr)
	})

	require.NoError(t, migrate.Up(ctx, dsn))
	_, err = pool.Exec(ctx, "TRUNCATE TABLE bookings, sessions, schedule_rules, schedules, slots, rooms, users")
	require.NoError(t, err)
	return &DB{Pool: pool, t: t}
}

func (db *DB) User() uuid.UUID {
	db.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), fixtureTimeout)
	defer cancel()
	id := uuid.New()
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO users (id, email, role, password_hash) VALUES ($1, $2, 'user', 'test-hash')",
		id, id.String()+"@example.test")
	require.NoError(db.t, err)
	return id
}

func (db *DB) Room() uuid.UUID {
	db.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), fixtureTimeout)
	defer cancel()
	id := uuid.New()
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO rooms (id, name, capacity) VALUES ($1, $2, 10)",
		id, "room-"+id.String())
	require.NoError(db.t, err)
	return id
}

func (db *DB) Slot(roomID uuid.UUID, start time.Time) uuid.UUID {
	db.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), fixtureTimeout)
	defer cancel()
	id := uuid.New()
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO slots (id, room_id, start_at, end_at) VALUES ($1, $2, $3, $4)",
		id, roomID, start.UTC(), start.Add(30*time.Minute).UTC())
	require.NoError(db.t, err)
	return id
}

func (db *DB) Booking(slotID, userID uuid.UUID, status string, createdAt time.Time) uuid.UUID {
	db.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), fixtureTimeout)
	defer cancel()
	id := uuid.New()
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO bookings (id, slot_id, user_id, status, created_at) VALUES ($1, $2, $3, $4::booking_status, $5)",
		id, slotID, userID, status, createdAt.UTC())
	require.NoError(db.t, err)
	return id
}
