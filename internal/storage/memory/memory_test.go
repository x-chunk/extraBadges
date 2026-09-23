package memory

import (
	"context"
	"errors"
	"sync"
	"testing"

	"extra-badges/internal/badge"
)

func TestRepository(t *testing.T) {
	ctx := context.Background()
	r := New(badge.Badge{UserID: 3, EmojiID: 30}, badge.Badge{UserID: 1, EmojiID: 10})

	if _, err := r.Get(ctx, 2); !errors.Is(err, badge.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if err := r.Create(ctx, badge.Badge{UserID: 1, EmojiID: 1}); !errors.Is(err, badge.ErrAlreadyExists) {
		t.Fatalf("want ErrAlreadyExists, got %v", err)
	}
	if err := r.Create(ctx, badge.Badge{UserID: 2, EmojiID: 20}); err != nil {
		t.Fatal(err)
	}
	list, _ := r.List(ctx)
	if len(list) != 3 || list[0].UserID != 1 || list[1].UserID != 2 || list[2].UserID != 3 {
		t.Fatalf("list not sorted/complete: %+v", list)
	}
	if err := r.Upsert(ctx, badge.Badge{UserID: 2, EmojiID: 99}); err != nil {
		t.Fatal(err)
	}
	if b, _ := r.Get(ctx, 2); b.EmojiID != 99 {
		t.Fatalf("upsert not applied: %+v", b)
	}
	if err := r.Delete(ctx, 2); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(ctx, 2); !errors.Is(err, badge.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestRepositoryConcurrent(t *testing.T) {
	ctx := context.Background()
	r := New()
	var wg sync.WaitGroup
	for i := range int64(100) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Upsert(ctx, badge.Badge{UserID: i + 1, EmojiID: i + 1})
			_, _ = r.Get(ctx, i+1)
			_, _ = r.List(ctx)
		}()
	}
	wg.Wait()
	if list, _ := r.List(ctx); len(list) != 100 {
		t.Fatalf("got %d badges, want 100", len(list))
	}
}
