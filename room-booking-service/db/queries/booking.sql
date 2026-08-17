-- name: CreateBooking :one
INSERT INTO bookings (slot_id,
                      user_id,
                      conference_link)
SELECT s.id,
       sqlc.arg(user_id),
       sqlc.narg(conference_link)
FROM slots AS s
WHERE s.id = sqlc.arg(slot_id)
  AND s.start_at >= NOW()
RETURNING
    id,
    slot_id,
    user_id,
    status,
    conference_link,
    created_at;

-- name: CancelBooking :execrows
UPDATE bookings
SET status = 'cancelled'
WHERE id = sqlc.arg(booking_id)
  AND user_id = sqlc.arg(user_id);

-- name: GetBookingByID :one
SELECT id,
       slot_id,
       user_id,
       status,
       conference_link,
       created_at
FROM bookings
WHERE id = sqlc.arg(booking_id);

-- name: ListBookings :many
SELECT id,
       slot_id,
       user_id,
       status,
       conference_link,
       created_at
FROM bookings
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: ListUserFutureBookings :many
SELECT b.id,
       b.slot_id,
       b.user_id,
       b.status,
       b.conference_link,
       b.created_at
FROM bookings AS b
         JOIN slots AS s
              ON b.slot_id = s.id
         JOIN rooms AS r
              ON r.id = s.room_id
WHERE b.user_id = sqlc.arg(user_id)
  AND s.start_at >= NOW()
ORDER BY s.start_at, b.id;

-- name: CountBookings :one
SELECT COUNT(*) AS total
FROM bookings;
