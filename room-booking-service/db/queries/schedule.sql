-- name: CreateSchedule :one
INSERT INTO schedules (room_id)
VALUES (sqlc.arg(room_id))
RETURNING
    id,
    room_id,
    created_at;

-- name: CreateScheduleRule :exec
INSERT INTO schedule_rules (schedule_id, day_of_week, start_at, end_at)
VALUES (sqlc.arg(schedule_id),
        sqlc.arg(day_of_week),
        sqlc.arg(start_at),
        sqlc.arg(end_at));

-- name: GetScheduleRuleForDay :one
SELECT s.id AS schedule_id,
       sr.day_of_week,
       sr.start_at,
       sr.end_at
FROM schedules AS s
         LEFT JOIN schedule_rules AS sr
                   ON sr.schedule_id = s.id
                       AND sr.day_of_week = sqlc.arg(day_of_week)
WHERE s.room_id = sqlc.arg(room_id)
;
