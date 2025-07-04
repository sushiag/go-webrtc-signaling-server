-- name: CreateUser :exec
INSERT INTO users (username, password)
VALUES (?, ?);

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
