-- +goose Up
CREATE TABLE schedule_rules
(
    schedule_id UUID    NOT NULL,
    day_of_week INTEGER NOT NULL,
    start_at    TIME    NOT NULL,
    end_at      TIME    NOT NULL,

    CONSTRAINT schedule_rules_unique_day
        PRIMARY KEY (schedule_id, day_of_week),

    CONSTRAINT schedule_rules_schedule_id_fk
        FOREIGN KEY (schedule_id) REFERENCES schedules (id) ON DELETE CASCADE,

    CONSTRAINT schedule_rules_day_valid
        CHECK (day_of_week BETWEEN 1 AND 7),

    CONSTRAINT schedule_rules_time_valid
        CHECK (start_at < end_at)
);

-- +goose Down
DROP TABLE IF EXISTS schedule_rules;
