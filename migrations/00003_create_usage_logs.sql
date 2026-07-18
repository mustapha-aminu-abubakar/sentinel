-- +goose Up
CREATE TABLE usage_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    api TEXT NOT NULL,
    allowed BOOLEAN NOT NULL,
    latency INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_usage_logs_lookup ON usage_logs (client_id, api, created_at);

-- +goose Down
DROP TABLE IF EXISTS usage_logs;
