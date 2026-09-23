// Package storagetest holds a behavioural test suite every badge.Repository must pass.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"extra-badges/internal/badge"
)

// Run exercises repo implementations built by newRepo (each call must return an empty store).
func Run(t *testing.T, newRepo func(t *testing.T) badge.Repository) {
	ctx := context.Background()

	t.Run("get missing", func(t *testing.T) {
		if _, err := newRepo(t).Get(ctx, 1); !errors.Is(err, badge.ErrNotFound) {
			t.Fatalf("want ErrNotFound, got %v", err)
		}
	})

	t.Run("create and get keeps int64 precision", func(t *testing.T) {
		r := newRepo(t)
		want := badge.Badge{UserID: 1256738876, EmojiID: 5276360730361229322, Title: "t"}
		if err := r.Create(ctx, want); err != nil {
			t.Fatal(err)
		}
		got, err := r.Get(ctx, want.UserID)
		if err != nil || got != want {
			t.Fatalf("Get = %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("create duplicate", func(t *testing.T) {
		r := newRepo(t)
		b := badge.Badge{UserID: 1, EmojiID: 1, Title: "a"}
		if err := r.Create(ctx, b); err != nil {
			t.Fatal(err)
		}
		b.EmojiID = 2
		if err := r.Create(ctx, b); !errors.Is(err, badge.ErrAlreadyExists) {
			t.Fatalf("want ErrAlreadyExists, got %v", err)
		}
		if got, _ := r.Get(ctx, 1); got.EmojiID != 1 {
			t.Fatalf("duplicate create overwrote data: %+v", got)
		}
	})

	t.Run("upsert inserts and replaces", func(t *testing.T) {
		r := newRepo(t)
		for _, emoji := range []int64{1, 2} {
			if err := r.Upsert(ctx, badge.Badge{UserID: 1, EmojiID: emoji, Title: "a"}); err != nil {
				t.Fatal(err)
			}
			if got, _ := r.Get(ctx, 1); got.EmojiID != emoji {
				t.Fatalf("after upsert emoji=%d got %+v", emoji, got)
			}
		}
	})

	t.Run("update", func(t *testing.T) {
		r := newRepo(t)
		if err := r.Update(ctx, badge.Badge{UserID: 1, EmojiID: 1, Title: "a"}); !errors.Is(err, badge.ErrNotFound) {
			t.Fatalf("update missing: want ErrNotFound, got %v", err)
		}
		_ = r.Create(ctx, badge.Badge{UserID: 1, EmojiID: 1, Title: "a"})
		_, _ = r.Get(ctx, 1) // warm any cache
		if err := r.Update(ctx, badge.Badge{UserID: 1, EmojiID: 5, Title: "b"}); err != nil {
			t.Fatal(err)
		}
		if got, _ := r.Get(ctx, 1); got != (badge.Badge{UserID: 1, EmojiID: 5, Title: "b"}) {
			t.Fatalf("stale after update: %+v", got)
		}
	})

	t.Run("delete", func(t *testing.T) {
		r := newRepo(t)
		_ = r.Create(ctx, badge.Badge{UserID: 1, EmojiID: 1, Title: "a"})
		_, _ = r.Get(ctx, 1)
		if err := r.Delete(ctx, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := r.Get(ctx, 1); !errors.Is(err, badge.ErrNotFound) {
			t.Fatalf("stale after delete: %v", err)
		}
		if err := r.Delete(ctx, 1); !errors.Is(err, badge.ErrNotFound) {
			t.Fatalf("second delete: want ErrNotFound, got %v", err)
		}
	})

	t.Run("list sorted, empty is non-nil", func(t *testing.T) {
		r := newRepo(t)
		list, err := r.List(ctx)
		if err != nil || list == nil || len(list) != 0 {
			t.Fatalf("empty list = %#v, %v", list, err)
		}
		for _, id := range []int64{3, 1, 2} {
			_ = r.Create(ctx, badge.Badge{UserID: id, EmojiID: id, Title: "a"})
		}
		list, _ = r.List(ctx)
		if len(list) != 3 || list[0].UserID != 1 || list[1].UserID != 2 || list[2].UserID != 3 {
			t.Fatalf("list = %+v", list)
		}
	})
}
