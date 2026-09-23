-- name: DatabaseTime :one
SELECT CURRENT_TIMESTAMP::timestamptz AS database_time;
