-- name: CreateUser :exec
INSERT INTO users (username, password, api_key)
VALUES (?, ?, ?);

-- name: UpdateUserPassword :exec
UPDATE users
SET password = ?
WHERE username = ?
AND password = ?;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = ?;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ?;

-- name: GetUserByApikeys :one
SELECT * FROM users WHERE api_key = ?;


-- name: UpdateAPIKey :exec
UPDATE users SET api_key = ? WHERE username = ? AND password = ?;