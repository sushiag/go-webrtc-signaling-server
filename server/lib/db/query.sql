-- name: CreateUser :exec
INSERT INTO users (username, password)
VALUES (?, ?);

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = ?;

-- name: UpdateUserPassword :exec
UPDATE users
SET password = ?
WHERE username = ? AND password = ?;