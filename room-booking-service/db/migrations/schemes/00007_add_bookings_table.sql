-- +goose Up
CREATE TYPE booking_status AS ENUM ('active', 'cancelled');

CREATE TABLE bookings
(
    id              UUID PRIMARY KEY        DEFAULT gen_random_uuid(),
    slot_id         UUID           NOT NULL,
    user_id         UUID           NOT NULL,
    status          booking_status NOT NULL DEFAULT 'active',
    conference_link TEXT,
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT bookings_slot_id_fk
        FOREIGN KEY (slot_id) REFERENCES slots (id) ON DELETE RESTRICT,

    CONSTRAINT bookings_user_id_fk
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX bookings_one_active_per_slot
    ON bookings (slot_id)
    WHERE status = 'active';

CREATE INDEX bookings_user_id_idx
    ON bookings (user_id);

CREATE INDEX bookings_created_at_id_idx
    ON bookings (created_at DESC, id DESC);

-- +goose Down
DROP TABLE IF EXISTS bookings;
DROP TYPE IF EXISTS booking_status;


