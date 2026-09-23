package cached

import (
	"context"
	"errors"
	"testing"
	"time"

	"extra-badges/internal/badge"
	"extra-badges/internal/storage/memory"
	"extra-badges/internal/storage/storagetest"
)

func TestContract(t *testing.T) {
	storagetest.Run(t, func(*testing.T) badge.Repository { return New(memory.New(), time.Minute) })
}

type countingRepo struct {
	badge.Repository
	gets int
}

func (c *countingRepo) Get(ctx context.Context, id int64) (badge.Badge, error) {
	c.gets++
	return c.Repository.Get(ctx, id)
}

func TestServesHitsFromCache(t *testing.T) {
	ctx := context.Background()
	backend := &countingRepo{Repository: memory.New(badge.Badge{UserID: 1, EmojiID: 1})}
	r := New(backend, time.Minute)

	for range 5 {
		if _, err := r.Get(ctx, 1); err != nil {
			t.Fatal(err)
		}
	}
	if backend.gets != 1 {
		t.Fatalf("backend hit %d times, want 1", backend.gets)
	}
}

func TestDoesNotCacheMisses(t *testing.T) {
	ctx := context.Background()
	backend := &countingRepo{Repository: memory.New()}
	r := New(backend, time.Minute)

	for range 3 {
		if _, err := r.Get(ctx, 1); !errors.Is(err, badge.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	}
	if backend.gets != 3 {
		t.Fatalf("backend hit %d times, want 3", backend.gets)
	}
}

func TestExpires(t *testing.T) {
	ctx := context.Background()
	backend := &countingRepo{Repository: memory.New(badge.Badge{UserID: 1, EmojiID: 1})}
	r := New(backend, time.Millisecond)

	_, _ = r.Get(ctx, 1)
	time.Sleep(5 * time.Millisecond)
	_, _ = r.Get(ctx, 1)
	if backend.gets != 2 {
		t.Fatalf("backend hit %d times, want 2 after TTL", backend.gets)
	}
}

func TestWriteBypassingCacheIsInvisibleUntilInvalidated(t *testing.T) {
	// Documents the trade-off: out-of-band writes are only seen after TTL.
	ctx := context.Background()
	backend := memory.New(badge.Badge{UserID: 1, EmojiID: 1})
	r := New(backend, time.Minute)

	_, _ = r.Get(ctx, 1)
	_ = backend.Upsert(ctx, badge.Badge{UserID: 1, EmojiID: 2})
	if got, _ := r.Get(ctx, 1); got.EmojiID != 1 {
		t.Fatalf("expected cached value, got %+v", got)
	}
	_ = r.Upsert(ctx, badge.Badge{UserID: 1, EmojiID: 3})
	if got, _ := r.Get(ctx, 1); got.EmojiID != 3 {
		t.Fatalf("expected fresh value after write, got %+v", got)
	}
}
