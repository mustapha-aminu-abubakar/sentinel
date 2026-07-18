-- name: CreateRateRule :one
INSERT INTO rate_rules (id, client_id, api, requests_allowed, window_seconds)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetRateRule :one
SELECT * FROM rate_rules WHERE id = $1;

-- name: GetRateRuleForUpdate :one
SELECT * FROM rate_rules WHERE id = $1 FOR UPDATE;

-- name: ListRateRulesByClient :many
SELECT * FROM rate_rules
WHERE client_id = $1
ORDER BY created_at DESC;

-- name: ListRateRules :many
SELECT * FROM rate_rules
WHERE (client_id = sqlc.narg('client_id') OR sqlc.narg('client_id') IS NULL)
  AND (api = sqlc.narg('api') OR sqlc.narg('api') IS NULL)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetRateRuleByClientAndAPI :one
SELECT * FROM rate_rules
WHERE client_id = $1 AND api = $2;

-- name: UpdateRateRule :one
UPDATE rate_rules
SET requests_allowed = $2, window_seconds = $3, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteRateRuleByClient :exec
DELETE FROM rate_rules WHERE id = $1 AND client_id = $2;
