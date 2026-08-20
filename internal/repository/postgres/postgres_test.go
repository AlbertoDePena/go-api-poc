package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/AlbertoDePena/go-api-poc/internal/domain"
	"github.com/AlbertoDePena/go-api-poc/internal/repository/postgres"
)

// newTestDB opens the PostgreSQL database named by TEST_DATABASE_URL and
// truncates the tables so each test starts from a clean slate. The whole
// suite is skipped when TEST_DATABASE_URL is unset, so `go test ./...` stays
// green in CI without a running database — these are integration tests.
func newTestDB(t *testing.T) *postgres.DB {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres integration tests")
	}

	db, err := postgres.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	truncate(t, url)
	return db
}

// truncate clears the tables via a throwaway connection so tests do not see
// rows left behind by earlier tests sharing the same database.
func truncate(t *testing.T, url string) {
	t.Helper()

	raw, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("open truncate conn: %v", err)
	}
	defer raw.Close()

	if _, err := raw.Exec("TRUNCATE greetings, outbox_messages"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// ---------- GreetingRepository ----------

func TestGreetingRepository_SaveAndFindByID(t *testing.T) {
	db := newTestDB(t)
	repo := postgres.NewGreetingRepository(db)
	ctx := context.Background()

	g := domain.NewGreeting("g-1", "Alice")

	if err := repo.Save(ctx, g); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID(ctx, "g-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	assertGreetingEqual(t, g, got)
}

func TestGreetingRepository_FindByID_NotFound(t *testing.T) {
	db := newTestDB(t)
	repo := postgres.NewGreetingRepository(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "does-not-exist")
	if !errors.Is(err, domain.ErrGreetingNotFound) {
		t.Fatalf("expected ErrGreetingNotFound, got %v", err)
	}
}

func TestGreetingRepository_SaveUpsert(t *testing.T) {
	db := newTestDB(t)
	repo := postgres.NewGreetingRepository(db)
	ctx := context.Background()

	g := domain.NewGreeting("g-1", "Alice")
	if err := repo.Save(ctx, g); err != nil {
		t.Fatalf("Save: %v", err)
	}

	g.Name = "Bob"
	g.Message = "Hello, Bob!"
	if err := repo.Save(ctx, g); err != nil {
		t.Fatalf("Save (upsert): %v", err)
	}

	got, err := repo.FindByID(ctx, "g-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	if got.Name != "Bob" {
		t.Errorf("Name = %q, want %q", got.Name, "Bob")
	}
	if got.Message != "Hello, Bob!" {
		t.Errorf("Message = %q, want %q", got.Message, "Hello, Bob!")
	}
}

func TestGreetingRepository_FindAll(t *testing.T) {
	db := newTestDB(t)
	repo := postgres.NewGreetingRepository(db)
	ctx := context.Background()

	now := time.Now()
	greetings := []*domain.Greeting{
		{ID: "g-1", Name: "Alice", Message: "Hello, Alice!", CreatedAt: now.Add(-2 * time.Second)},
		{ID: "g-2", Name: "Bob", Message: "Hello, Bob!", CreatedAt: now.Add(-1 * time.Second)},
		{ID: "g-3", Name: "Carol", Message: "Hello, Carol!", CreatedAt: now},
	}

	for _, g := range greetings {
		if err := repo.Save(ctx, g); err != nil {
			t.Fatalf("Save %s: %v", g.ID, err)
		}
	}

	got, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("FindAll returned %d rows, want 3", len(got))
	}

	// ORDER BY created_at DESC — newest first
	if got[0].ID != "g-3" || got[1].ID != "g-2" || got[2].ID != "g-1" {
		t.Errorf("FindAll order = [%s, %s, %s], want [g-3, g-2, g-1]",
			got[0].ID, got[1].ID, got[2].ID)
	}
}

func TestGreetingRepository_FindAll_Empty(t *testing.T) {
	db := newTestDB(t)
	repo := postgres.NewGreetingRepository(db)
	ctx := context.Background()

	got, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("FindAll returned %d rows, want 0", len(got))
	}
}

// ---------- OutboxRepository ----------

func TestOutboxRepository_SaveAndFindPending(t *testing.T) {
	db := newTestDB(t)
	repo := postgres.NewOutboxRepository(db)
	ctx := context.Background()

	msg := domain.NewOutboxMessage("m-1", "Greeting", "g-1", "GreetingCreated", []byte(`{"name":"Alice"}`))

	if err := repo.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	pending, err := repo.FindPending(ctx, 10)
	if err != nil {
		t.Fatalf("FindPending: %v", err)
	}

	if len(pending) != 1 {
		t.Fatalf("FindPending returned %d, want 1", len(pending))
	}

	got := pending[0]
	if got.ID != msg.ID {
		t.Errorf("ID = %q, want %q", got.ID, msg.ID)
	}
	if got.AggregateType != "Greeting" {
		t.Errorf("AggregateType = %q, want %q", got.AggregateType, "Greeting")
	}
	if got.EventType != "GreetingCreated" {
		t.Errorf("EventType = %q, want %q", got.EventType, "GreetingCreated")
	}
	if string(got.Payload) != `{"name":"Alice"}` {
		t.Errorf("Payload = %q, want %q", string(got.Payload), `{"name":"Alice"}`)
	}
	if got.ProcessedAt != nil {
		t.Errorf("ProcessedAt = %v, want nil", got.ProcessedAt)
	}
}

func TestOutboxRepository_FindPending_RespectsLimit(t *testing.T) {
	db := newTestDB(t)
	repo := postgres.NewOutboxRepository(db)
	ctx := context.Background()

	for i := range 5 {
		msg := domain.NewOutboxMessage(
			fmt.Sprintf("m-%d", i), "Greeting", fmt.Sprintf("g-%d", i),
			"GreetingCreated", []byte(`{}`),
		)
		msg.CreatedAt = time.Now().Add(time.Duration(i) * time.Second)
		if err := repo.Save(ctx, msg); err != nil {
			t.Fatalf("Save m-%d: %v", i, err)
		}
	}

	pending, err := repo.FindPending(ctx, 3)
	if err != nil {
		t.Fatalf("FindPending: %v", err)
	}

	if len(pending) != 3 {
		t.Fatalf("FindPending returned %d, want 3", len(pending))
	}
}

func TestOutboxRepository_FindPending_OrderByCreatedAtASC(t *testing.T) {
	db := newTestDB(t)
	repo := postgres.NewOutboxRepository(db)
	ctx := context.Background()

	now := time.Now()
	for i := range 3 {
		msg := &domain.OutboxMessage{
			ID:            fmt.Sprintf("m-%d", i),
			AggregateType: "Greeting",
			AggregateID:   fmt.Sprintf("g-%d", i),
			EventType:     "GreetingCreated",
			Payload:       []byte(`{}`),
			CreatedAt:     now.Add(time.Duration(i) * time.Second),
		}
		if err := repo.Save(ctx, msg); err != nil {
			t.Fatalf("Save m-%d: %v", i, err)
		}
	}

	pending, err := repo.FindPending(ctx, 10)
	if err != nil {
		t.Fatalf("FindPending: %v", err)
	}

	// Oldest first
	if pending[0].ID != "m-0" || pending[1].ID != "m-1" || pending[2].ID != "m-2" {
		t.Errorf("order = [%s, %s, %s], want [m-0, m-1, m-2]",
			pending[0].ID, pending[1].ID, pending[2].ID)
	}
}

func TestOutboxRepository_MarkProcessed(t *testing.T) {
	db := newTestDB(t)
	repo := postgres.NewOutboxRepository(db)
	ctx := context.Background()

	msg := domain.NewOutboxMessage("m-1", "Greeting", "g-1", "GreetingCreated", []byte(`{}`))
	if err := repo.Save(ctx, msg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.MarkProcessed(ctx, "m-1"); err != nil {
		t.Fatalf("MarkProcessed: %v", err)
	}

	pending, err := repo.FindPending(ctx, 10)
	if err != nil {
		t.Fatalf("FindPending: %v", err)
	}

	if len(pending) != 0 {
		t.Errorf("FindPending returned %d after MarkProcessed, want 0", len(pending))
	}
}

// ---------- Transactor ----------

func TestTransactor_CommitsOnSuccess(t *testing.T) {
	db := newTestDB(t)
	uow := postgres.NewTransactor(db)
	greetingRepo := postgres.NewGreetingRepository(db)
	ctx := context.Background()

	g := domain.NewGreeting("g-1", "Alice")

	err := uow.WithinTx(ctx, func(ctx context.Context) error {
		return greetingRepo.Save(ctx, g)
	})
	if err != nil {
		t.Fatalf("WithinTx: %v", err)
	}

	got, err := greetingRepo.FindByID(ctx, "g-1")
	if err != nil {
		t.Fatalf("FindByID after commit: %v", err)
	}
	assertGreetingEqual(t, g, got)
}

func TestTransactor_RollsBackOnError(t *testing.T) {
	db := newTestDB(t)
	uow := postgres.NewTransactor(db)
	greetingRepo := postgres.NewGreetingRepository(db)
	ctx := context.Background()

	deliberate := errors.New("deliberate error")

	err := uow.WithinTx(ctx, func(ctx context.Context) error {
		g := domain.NewGreeting("g-1", "Alice")
		if err := greetingRepo.Save(ctx, g); err != nil {
			return err
		}
		return deliberate
	})
	if !errors.Is(err, deliberate) {
		t.Fatalf("WithinTx error = %v, want deliberate error", err)
	}

	_, err = greetingRepo.FindByID(ctx, "g-1")
	if !errors.Is(err, domain.ErrGreetingNotFound) {
		t.Fatalf("expected ErrGreetingNotFound after rollback, got %v", err)
	}
}

func TestTransactor_RollsBackOnPanic(t *testing.T) {
	db := newTestDB(t)
	uow := postgres.NewTransactor(db)
	greetingRepo := postgres.NewGreetingRepository(db)
	ctx := context.Background()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic to propagate")
		}
		if r != "boom" {
			t.Fatalf("recovered %v, want %q", r, "boom")
		}

		// Verify rollback happened
		_, err := greetingRepo.FindByID(ctx, "g-1")
		if !errors.Is(err, domain.ErrGreetingNotFound) {
			t.Fatalf("expected ErrGreetingNotFound after panic rollback, got %v", err)
		}
	}()

	_ = uow.WithinTx(ctx, func(ctx context.Context) error {
		g := domain.NewGreeting("g-1", "Alice")
		if err := greetingRepo.Save(ctx, g); err != nil {
			return err
		}
		panic("boom")
	})
}

func TestTransactor_NestedReusesTx(t *testing.T) {
	db := newTestDB(t)
	uow := postgres.NewTransactor(db)
	greetingRepo := postgres.NewGreetingRepository(db)
	ctx := context.Background()

	err := uow.WithinTx(ctx, func(ctx context.Context) error {
		g := domain.NewGreeting("g-1", "Alice")
		if err := greetingRepo.Save(ctx, g); err != nil {
			return err
		}
		// Nested WithinTx should reuse the existing tx
		return uow.WithinTx(ctx, func(ctx context.Context) error {
			g2 := domain.NewGreeting("g-2", "Bob")
			return greetingRepo.Save(ctx, g2)
		})
	})
	if err != nil {
		t.Fatalf("nested WithinTx: %v", err)
	}

	for _, id := range []string{"g-1", "g-2"} {
		if _, err := greetingRepo.FindByID(ctx, id); err != nil {
			t.Errorf("FindByID(%s) after nested WithinTx: %v", id, err)
		}
	}
}

// ---------- Transactor + Multiple Repositories (Integration) ----------

func TestTransactor_AtomicCommit_GreetingAndOutbox(t *testing.T) {
	db := newTestDB(t)
	uow := postgres.NewTransactor(db)
	greetingRepo := postgres.NewGreetingRepository(db)
	outboxRepo := postgres.NewOutboxRepository(db)
	ctx := context.Background()

	g := domain.NewGreeting("g-1", "Alice")
	msg := domain.NewOutboxMessage("m-1", "Greeting", "g-1", "GreetingCreated", []byte(`{"name":"Alice"}`))

	err := uow.WithinTx(ctx, func(ctx context.Context) error {
		if err := greetingRepo.Save(ctx, g); err != nil {
			return err
		}
		return outboxRepo.Save(ctx, msg)
	})
	if err != nil {
		t.Fatalf("WithinTx: %v", err)
	}

	got, err := greetingRepo.FindByID(ctx, "g-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	assertGreetingEqual(t, g, got)

	pending, err := outboxRepo.FindPending(ctx, 10)
	if err != nil {
		t.Fatalf("FindPending: %v", err)
	}
	if len(pending) != 1 || pending[0].ID != "m-1" {
		t.Fatalf("expected 1 pending outbox message with ID m-1, got %d", len(pending))
	}
}

func TestTransactor_AtomicRollback_GreetingAndOutbox(t *testing.T) {
	db := newTestDB(t)
	uow := postgres.NewTransactor(db)
	greetingRepo := postgres.NewGreetingRepository(db)
	outboxRepo := postgres.NewOutboxRepository(db)
	ctx := context.Background()

	deliberate := errors.New("deliberate error")

	err := uow.WithinTx(ctx, func(ctx context.Context) error {
		g := domain.NewGreeting("g-1", "Alice")
		if err := greetingRepo.Save(ctx, g); err != nil {
			return err
		}
		msg := domain.NewOutboxMessage("m-1", "Greeting", "g-1", "GreetingCreated", []byte(`{}`))
		if err := outboxRepo.Save(ctx, msg); err != nil {
			return err
		}
		return deliberate
	})
	if !errors.Is(err, deliberate) {
		t.Fatalf("WithinTx error = %v, want deliberate error", err)
	}

	_, err = greetingRepo.FindByID(ctx, "g-1")
	if !errors.Is(err, domain.ErrGreetingNotFound) {
		t.Fatalf("expected ErrGreetingNotFound after rollback, got %v", err)
	}

	pending, err := outboxRepo.FindPending(ctx, 10)
	if err != nil {
		t.Fatalf("FindPending: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending messages after rollback, got %d", len(pending))
	}
}

// ---------- DB ----------

func TestDB_Ping(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestDB_Open_InvalidURL(t *testing.T) {
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping postgres integration tests")
	}
	_, err := postgres.Open(context.Background(), "postgres://nobody:nobody@127.0.0.1:1/nonexistent?sslmode=disable&connect_timeout=1")
	if err == nil {
		t.Fatal("expected error for unreachable database")
	}
}

// ---------- helpers ----------

func assertGreetingEqual(t *testing.T, want, got *domain.Greeting) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Message != want.Message {
		t.Errorf("Message = %q, want %q", got.Message, want.Message)
	}
	// Postgres TIMESTAMPTZ stores microsecond precision; truncate both sides
	// before comparing so the round-trip does not fail on dropped nanoseconds.
	if !got.CreatedAt.Truncate(time.Microsecond).Equal(want.CreatedAt.Truncate(time.Microsecond)) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, want.CreatedAt)
	}
}
