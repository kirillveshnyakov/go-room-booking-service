-- +goose Up
CREATE TABLE schedules
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    room_id    UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT schedules_room_id_unique
        UNIQUE (room_id),

    CONSTRAINT schedules_room_id_fk
        FOREIGN KEY (room_id) REFERENCES rooms (id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS schedules;
