-- +goose Up
CREATE TABLE rooms
(
    id          UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    capacity    INTEGER     NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT rooms_name_unique
        UNIQUE (name),

    CONSTRAINT rooms_capacity_positive
        CHECK (capacity > 0)
);

-- +goose Down
DROP TABLE IF EXISTS rooms;
