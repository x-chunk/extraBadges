package badge_test

import (
	"context"
	"errors"
	"testing"

	"extra-badges/internal/badge"
	"extra-badges/internal/storage/memory"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		in      badge.Badge
		wantErr bool
		title   string
	}{
		{"ok", badge.Badge{UserID: 1, EmojiID: 2, Title: " hi "}, false, "hi"},
		{"default title", badge.Badge{UserID: 1, EmojiID: 2}, false, badge.DefaultTitle},
		{"zero user", badge.Badge{EmojiID: 2}, true, ""},
		{"negative emoji", badge.Badge{UserID: 1, EmojiID: -1}, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.in
			err := b.Validate()
			if tt.wantErr {
				if !errors.Is(err, badge.ErrInvalid) {
					t.Fatalf("want ErrInvalid, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if b.Title != tt.title {
				t.Errorf("title = %q, want %q", b.Title, tt.title)
			}
		})
	}
}

func TestService(t *testing.T) {
	ctx := context.Background()
	svc := badge.NewService(memory.New())

	if _, err := svc.Get(ctx, 0); !errors.Is(err, badge.ErrInvalid) {
		t.Fatalf("Get(0): want ErrInvalid, got %v", err)
	}
	if _, err := svc.Create(ctx, badge.Badge{UserID: 1}); !errors.Is(err, badge.ErrInvalid) {
		t.Fatalf("Create invalid: want ErrInvalid, got %v", err)
	}
	created, err := svc.Create(ctx, badge.Badge{UserID: 1, EmojiID: 10})
	if err != nil {
		t.Fatal(err)
	}
	if created.Title != badge.DefaultTitle {
		t.Errorf("default title not applied: %q", created.Title)
	}
	if _, err := svc.Create(ctx, badge.Badge{UserID: 1, EmojiID: 11}); !errors.Is(err, badge.ErrAlreadyExists) {
		t.Fatalf("duplicate: want ErrAlreadyExists, got %v", err)
	}
	if _, err := svc.Put(ctx, badge.Badge{UserID: 1, EmojiID: 12, Title: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Put(ctx, badge.Badge{UserID: 1}); !errors.Is(err, badge.ErrInvalid) {
		t.Fatalf("Put invalid: want ErrInvalid, got %v", err)
	}
	got, err := svc.Get(ctx, 1)
	if err != nil || got.EmojiID != 12 || got.Title != "x" {
		t.Fatalf("Get after Put = %+v, %v", got, err)
	}
	list, err := svc.List(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("List = %v, %v", list, err)
	}
	if err := svc.Delete(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, 1); !errors.Is(err, badge.ErrNotFound) {
		t.Fatalf("after delete: want ErrNotFound, got %v", err)
	}
}

func ptr[T any](v T) *T { return &v }

func TestServiceUpdate(t *testing.T) {
	ctx := context.Background()
	svc := badge.NewService(memory.New(badge.Badge{UserID: 1, EmojiID: 10, Title: "old"}))

	if _, err := svc.Update(ctx, 2, badge.Patch{EmojiID: ptr[int64](1)}); !errors.Is(err, badge.ErrNotFound) {
		t.Fatalf("missing: want ErrNotFound, got %v", err)
	}
	got, err := svc.Update(ctx, 1, badge.Patch{Title: ptr("new")})
	if err != nil || got != (badge.Badge{UserID: 1, EmojiID: 10, Title: "new"}) {
		t.Fatalf("title patch = %+v, %v", got, err)
	}
	got, err = svc.Update(ctx, 1, badge.Patch{EmojiID: ptr[int64](20)})
	if err != nil || got.EmojiID != 20 || got.Title != "new" {
		t.Fatalf("emoji patch = %+v, %v", got, err)
	}
	if _, err := svc.Update(ctx, 1, badge.Patch{EmojiID: ptr[int64](-1)}); !errors.Is(err, badge.ErrInvalid) {
		t.Fatalf("invalid: want ErrInvalid, got %v", err)
	}
	if got, _ := svc.Get(ctx, 1); got.EmojiID != 20 {
		t.Fatalf("invalid patch was persisted: %+v", got)
	}
}
