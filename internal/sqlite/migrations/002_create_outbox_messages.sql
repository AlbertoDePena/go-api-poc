CREATE TABLE IF NOT EXISTS outbox_messages (
    id             TEXT PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id   TEXT NOT NULL,
    event_type     TEXT NOT NULL,
    payload        BLOB NOT NULL,
    created_at     TEXT NOT NULL,
    processed_at   TEXT
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending
    ON outbox_messages (created_at)
    WHERE processed_at IS NULL;
