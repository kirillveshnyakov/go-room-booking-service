-- +goose Up
CREATE TABLE slots
(
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id  UUID        NOT NULL,
    start_at TIMESTAMPTZ NOT NULL,
    end_at   TIMESTAMPTZ NOT NULL,

    CONSTRAINT slots_room_id_fk
        FOREIGN KEY (room_id) REFERENCES rooms (id) ON DELETE RESTRICT,

    CONSTRAINT slots_duration_valid
        CHECK (end_at = start_at + INTERVAL '30 minutes'),

    CONSTRAINT slots_room_start_unique
        UNIQUE (room_id, start_at),

    CONSTRAINT slots_no_overlap
        EXCLUDE USING GIST (
        room_id WITH =,
        tstzrange(start_at, end_at, '[)') WITH &&
        )
);

-- +goose Down
DROP TABLE IF EXISTS slots;
