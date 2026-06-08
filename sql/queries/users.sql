-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password, is_chirpy_red)
VALUES (
    gen_random_uuid(),
    now(),
    now(),
    $1,
    $2,
    FALSE
)
RETURNING id, created_at, updated_at, email, is_chirpy_red;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUser :one
SELECT id, created_at, updated_at, email, is_chirpy_red, hashed_password from users where email = $1;

-- name: UpdateUser :one
UPDATE users
SET updated_at = now(), email = $2, hashed_password = $3
WHERE id = $1
RETURNING id, created_at, updated_at, email, hashed_password;

-- name: UpgradeUser :exec
UPDATE users
SET is_chirpy_red = TRUE
where id = $1;
