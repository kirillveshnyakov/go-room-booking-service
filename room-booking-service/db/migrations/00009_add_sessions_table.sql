-- +goose Up

CREATE TABLE sessions
(
    id                 UUID PRIMARY KEY,
    user_id            UUID        NOT NULL,
    refresh_token_hash BYTEA       NOT NULL,
    expires_at         TIMESTAMPTZ NOT NULL,
    revoked_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT sessions_user_id_fk
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX sessions_user_id_idx
    ON sessions (user_id);

CREATE TRIGGER sessions_set_updated_at
    BEFORE UPDATE
    ON sessions
    FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TRIGGER IF EXISTS sessions_set_updated_at ON sessions;
DROP TABLE IF EXISTS sessions;
