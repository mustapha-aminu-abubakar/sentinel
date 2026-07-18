-- +goose Up
CREATE TABLE rate_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    api TEXT NOT NULL,
    requests_allowed INTEGER NOT NULL CHECK (requests_allowed > 0),
    window_seconds INTEGER NOT NULL CHECK (window_seconds > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (client_id, api)
);

-- +goose Down
DROP TABLE rate_rules;
