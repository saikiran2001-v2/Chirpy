-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    now(),
    now(),
    $1,
    $2
)
RETURNING id, created_at, updated_at, email;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUser :one
SELECT id, created_at, updated_at, email, hashed_password from users where email = $1;

-- name: UpdateUser :one
UPDATE users
SET updated_at = now(), email = $2, hashed_password = $3
WHERE id = $1
RETURNING id, created_at, updated_at, email, hashed_password;
