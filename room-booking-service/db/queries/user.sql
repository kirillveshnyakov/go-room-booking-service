-- name: CreateUser :one
INSERT INTO users (email, role, password_hash)
VALUES (sqlc.arg(email),
        sqlc.arg(role),
        sqlc.arg(password_hash))
RETURNING
    id,
    email,
    role,
    created_at;

-- name: GetUserByEmail :one
SELECT id,
       email,
       role,
       password_hash,
       created_at
FROM users
WHERE email = sqlc.arg(email);
