-- name: CreateSlotsForDate :execrows
INSERT INTO slots (room_id,
                   start_at,
                   end_at)
SELECT sqlc.arg(room_id),
       generated_slot.start_at,
       generated_slot.end_at
FROM unnest(
             sqlc.arg(start_ats)::timestamptz[],
             sqlc.arg(end_ats)::timestamptz[]
     ) AS generated_slot (
                          start_at,
                          end_at
    )
ON CONFLICT (room_id, start_at)
    DO NOTHING;

-- name: ListFreeSlots :many
WITH requested_date AS (SELECT sqlc.arg(target_date)::date AS value)
SELECT s.id, s.room_id, s.start_at, s.end_at
FROM slots AS s
         CROSS JOIN requested_date AS rd
WHERE s.room_id = sqlc.arg(room_id)
  AND s.start_at >= (rd.value::timestamp AT TIME ZONE 'UTC')
  AND s.start_at < ((rd.value + 1)::timestamp AT TIME ZONE 'UTC')
  AND s.start_at > NOW()
  AND NOT EXISTS (SELECT 1
                  FROM bookings AS b
                  WHERE b.slot_id = s.id
                    AND b.status = 'active')
ORDER BY s.start_at, s.id;

-- name: GetSlotByID :one
SELECT id, room_id, start_at, end_at
FROM slots
WHERE id = sqlc.arg(slot_id);