package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"

	"extra-badges/internal/badge"
	"extra-badges/internal/client"
	"extra-badges/internal/storage/storagetest"
)

func open(t *testing.T, path string) *Repository {
	t.Helper()
	r, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

func TestContract(t *testing.T) {
	storagetest.Run(t, func(t *testing.T) badge.Repository {
		return open(t, filepath.Join(t.TempDir(), "test.db"))
	})
}

func TestPersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	r, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	want := badge.Badge{UserID: 1, EmojiID: 2, Description: "x"}
	if err := r.Create(ctx, want); err != nil {
		t.Fatal(err)
	}
	r.Close()

	got, err := open(t, path).Get(ctx, 1)
	if err != nil || got != want {
		t.Fatalf("after reopen: %+v, %v", got, err)
	}
}

func TestConcurrentWrites(t *testing.T) {
	ctx := context.Background()
	r := open(t, filepath.Join(t.TempDir(), "test.db"))
	var wg sync.WaitGroup
	for i := range int64(50) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := r.Upsert(ctx, badge.Badge{UserID: i + 1, EmojiID: 1, Description: "a"}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if list, _ := r.List(ctx); len(list) != 50 {
		t.Fatalf("got %d rows", len(list))
	}
}

func TestOpenBadPath(t *testing.T) {
	if _, err := Open(context.Background(), filepath.Join(t.TempDir(), "missing", "dir", "x.db")); err == nil {
		t.Fatal("expected error for unwritable path")
	}
}

func TestClosedDBErrors(t *testing.T) {
	ctx := context.Background()
	r, err := Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	r.Close()
	b := badge.Badge{UserID: 1, EmojiID: 1, Description: "a"}
	checks := map[string]error{
		"get":    func() error { _, err := r.Get(ctx, 1); return err }(),
		"list":   func() error { _, err := r.List(ctx); return err }(),
		"many":   func() error { _, err := r.GetMany(ctx, []int64{1}); return err }(),
		"create": r.Create(ctx, b),
		"upsert": r.Upsert(ctx, b),
		"update": r.Update(ctx, b),
		"delete": r.Delete(ctx, 1),
	}
	for name, err := range checks {
		if err == nil {
			t.Errorf("%s on closed db: expected error", name)
		}
	}
}

func TestMigratesLegacySchema(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "legacy.db")
	raw, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, migrations[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO badges (user_id, emoji_id, title) VALUES (1, 2, 'old')`); err != nil {
		t.Fatal(err)
	}
	raw.Close()

	got, err := open(t, path).Get(ctx, 1)
	if err != nil || got != (badge.Badge{UserID: 1, EmojiID: 2, Description: "old"}) {
		t.Fatalf("after migration: %+v, %v", got, err)
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	open(t, path).Close()
	open(t, path)
}

func TestClientsContract(t *testing.T) {
	storagetest.RunClients(t, func(t *testing.T) client.Repository {
		return open(t, filepath.Join(t.TempDir(), "test.db")).Clients()
	})
}
