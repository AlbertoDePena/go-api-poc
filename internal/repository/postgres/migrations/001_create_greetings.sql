CREATE TABLE IF NOT EXISTS greetings (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    message    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
