-- name: CreateRoom :one
INSERT INTO rooms (name,
                   description,
                   capacity)
VALUES (sqlc.arg(name),
        sqlc.arg(description),
        sqlc.arg(capacity))
    RETURNING
    id,
    name,
    description,
    capacity,
    created_at;

-- name: GetRoomByID :one
SELECT id, name, description, capacity, created_at
FROM rooms
WHERE id = sqlc.arg(room_id);

-- name: ListRooms :many
SELECT id, name, description, capacity, created_at
FROM rooms
ORDER BY created_at, id;