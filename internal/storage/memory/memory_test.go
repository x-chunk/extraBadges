package memory

import (
	"context"
	"sync"
	"testing"

	"extra-badges/internal/badge"
	"extra-badges/internal/storage/storagetest"
)

func TestContract(t *testing.T) {
	storagetest.Run(t, func(*testing.T) badge.Repository { return New() })
}

func TestSeed(t *testing.T) {
	r := New(badge.Badge{UserID: 1, EmojiID: 10})
	if b, err := r.Get(context.Background(), 1); err != nil || b.EmojiID != 10 {
		t.Fatalf("seed not loaded: %+v %v", b, err)
	}
}

func TestConcurrent(t *testing.T) {
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
