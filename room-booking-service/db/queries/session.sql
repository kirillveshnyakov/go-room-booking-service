-- name: CreateSession :one
INSERT INTO sessions (id, user_id, refresh_token_hash, expires_at)
VALUES (sqlc.arg(id),
        sqlc.arg(user_id),
        sqlc.arg(refresh_token_hash),
        sqlc.arg(expires_at))
RETURNING
    id,
    user_id,
    refresh_token_hash,
    expires_at,
    revoked_at,
    created_at,
    updated_at;

-- name: GetByID :one
SELECT id, user_id, refresh_token_hash, expires_at, revoked_at, created_at, updated_at
FROM sessions
WHERE id = sqlc.arg(id);

-- name: Revoke :exec
UPDATE sessions
SET revoked_at = sqlc.arg(revoked_at)
WHERE id = sqlc.arg(id)
  AND revoked_at IS NULL;

-- name: RevokeAllByUser :exec
UPDATE sessions
SET revoked_at = sqlc.arg(revoked_at)
WHERE user_id = sqlc.arg(user_id)
  AND revoked_at IS NULL;

-- name: RotateRefreshToken :one
UPDATE sessions
SET refresh_token_hash = sqlc.arg(new_refresh_token_hash)
WHERE id = sqlc.arg(id)
  AND refresh_token_hash = sqlc.arg(old_refresh_token_hash)
  AND revoked_at IS NULL
  AND expires_at > NOW()
RETURNING
    id,
    user_id,
    refresh_token_hash,
    expires_at,
    revoked_at,
    created_at,
    updated_at;
