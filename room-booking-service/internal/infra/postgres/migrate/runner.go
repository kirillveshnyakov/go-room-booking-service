package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/db/migrations"
)

func Up(ctx context.Context, dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("migration runner - open postgres: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err = db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("migration runner - ping postgres: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		migrations.FS,
	)
	if err != nil {
		return fmt.Errorf("migration runner - create goose provider: %w", err)
	}

	if _, err = provider.Up(ctx); err != nil {
		return fmt.Errorf("migration runner - migrations up: %w", err)
	}

	return nil
}
