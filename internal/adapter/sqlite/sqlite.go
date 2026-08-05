package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/AlbertoDePena/go-api-poc/internal/adapter/sqlite/migrations"
	_ "modernc.org/sqlite"
)

// DB holds the read and write connection pools for SQLite.
// Separating reads from writes lets WAL mode serve many concurrent
// readers while a single writer avoids SQLITE_BUSY contention.
type DB struct {
	readDB  *sql.DB
	writeDB *sql.DB
}

// Open creates a new DB backed by the SQLite file at databasePath.
// It configures WAL mode, busy timeout, and other PRAGMAs internally,
// then runs all pending schema migrations before returning.
func Open(databasePath string) (*DB, error) {
	// SQLite will not create missing parent directories; opening a file in
	// one fails deep inside the driver with an opaque error. Check up front
	// and return an actionable message instead.
	if dir := filepath.Dir(databasePath); dir != "" && dir != "." {
		if _, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("database directory %q does not exist (create it, e.g. `mkdir -p %s`, or fix DATABASE_PATH)", dir, dir)
			}
			return nil, fmt.Errorf("stat database dir %q: %w", dir, err)
		}
	}

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

	if err := runMigrations(writeDB); err != nil {
		readDB.Close()
		writeDB.Close()
		return nil, err
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

func runMigrations(db *sql.DB) error {
	const createTracker = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`
	if _, err := db.Exec(createTracker); err != nil {
		return fmt.Errorf("create migration tracker: %w", err)
	}

	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		var exists int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM schema_migrations WHERE filename = ?",
			entry.Name(),
		).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if exists > 0 {
			continue
		}

		content, err := fs.ReadFile(migrations.FS, entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("run migration %s: %w", entry.Name(), err)
		}

		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (filename, applied_at) VALUES (?, ?)",
			entry.Name(), time.Now().Format(time.RFC3339Nano),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}

		slog.Info("applied migration", "file", entry.Name())
	}

	return nil
}
