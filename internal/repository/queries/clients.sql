-- name: CreateClient :one
INSERT INTO clients (id, name, status)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetClient :one
SELECT * FROM clients WHERE id = $1;

-- name: GetClientForUpdate :one
SELECT * FROM clients WHERE id = $1 FOR UPDATE;

-- name: ListClients :many
SELECT * FROM clients
WHERE (status = sqlc.narg('status') OR sqlc.narg('status') IS NULL)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: UpdateClient :one
UPDATE clients
SET name = $2, status = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeactivateClient :one
UPDATE clients
SET status = 'inactive', updated_at = now()
WHERE id = $1
RETURNING *;
