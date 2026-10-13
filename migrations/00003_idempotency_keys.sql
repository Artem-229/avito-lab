-- +goose Up
CREATE TABLE idempotency_keys
(
    key          UUID PRIMARY KEY,
    request_hash TEXT        NOT NULL,
    trip_id      UUID REFERENCES trips (id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE idempotency_keys;
