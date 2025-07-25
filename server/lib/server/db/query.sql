-- name: CreateUser :exec
INSERT INTO users (username, password, api_key)
VALUES (?, ?, ?);

-- name: UpdateUserPassword :exec
UPDATE users
SET password = ?, updated_at = CURRENT_TIMESTAMP 
WHERE username = ?;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = ?;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ? LIMIT 1;

-- name: GetUserByApikeys :one
SELECT * FROM users WHERE api_key = ?;

-- name: UpdateAPIKey :exec
UPDATE users
SET api_key = ?, updated_at = CURRENT_TIMESTAMP
WHERE username = ?;