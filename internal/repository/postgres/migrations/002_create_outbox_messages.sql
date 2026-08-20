CREATE TABLE IF NOT EXISTS outbox_messages (
    id             TEXT PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id   TEXT NOT NULL,
    event_type     TEXT NOT NULL,
    payload        BYTEA NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL,
    processed_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending
    ON outbox_messages (created_at)
    WHERE processed_at IS NULL;
