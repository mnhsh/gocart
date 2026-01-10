-- name: Reset :exec
TRUNCATE TABLE
    refresh_tokens,
    users
RESTART IDENTITY CASCADE;
