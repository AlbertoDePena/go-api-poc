package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"runtime"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS greetings (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    message    TEXT NOT NULL,
    created_at TEXT NOT NULL
);`

// DB holds the read and write connection pools for SQLite.
// Separating reads from writes lets WAL mode serve many concurrent
// readers while a single writer avoids SQLITE_BUSY contention.
type DB struct {
	readDB  *sql.DB
	writeDB *sql.DB
}

// Open creates a new DB backed by the SQLite file at databasePath.
// It configures WAL mode, busy timeout, and other PRAGMAs internally,
// then runs the schema migration before returning.
func Open(databasePath string) (*DB, error) {
	writeDSN := buildDSN(databasePath, false)
	readDSN := buildDSN(databasePath, true)

	writeDB, err := sql.Open("sqlite", writeDSN)
	if err != nil {
		return nil, fmt.Errorf("open write db: %w", err)
	}
	writeDB.SetMaxOpenConns(1)
	writeDB.SetMaxIdleConns(1)
	writeDB.SetConnMaxLifetime(0)

	readDB, err := sql.Open("sqlite", readDSN)
	if err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("open read db: %w", err)
	}
	maxReaders := max(4, runtime.NumCPU())
	readDB.SetMaxOpenConns(maxReaders)
	readDB.SetMaxIdleConns(maxReaders)
	readDB.SetConnMaxLifetime(0)

	if _, err := writeDB.Exec(schema); err != nil {
		readDB.Close()
		writeDB.Close()
		return nil, fmt.Errorf("run schema migration: %w", err)
	}

	return &DB{readDB: readDB, writeDB: writeDB}, nil
}

// Close closes both the read and write connection pools.
func (db *DB) Close() error {
	return errors.Join(db.readDB.Close(), db.writeDB.Close())
}

// Ping verifies that the database is reachable.
func (db *DB) Ping(ctx context.Context) error {
	return errors.Join(db.readDB.PingContext(ctx), db.writeDB.PingContext(ctx))
}

func buildDSN(path string, readOnly bool) string {
	params := url.Values{}
	params.Set("_pragma", "busy_timeout(5000)")
	params.Add("_pragma", "journal_mode(WAL)")
	params.Add("_pragma", "synchronous(NORMAL)")
	params.Add("_pragma", "cache_size(-64000)")
	params.Add("_pragma", "foreign_keys(ON)")
	if readOnly {
		params.Set("mode", "ro")
	}
	return fmt.Sprintf("file:%s?%s", path, params.Encode())
}
