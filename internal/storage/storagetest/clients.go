package storagetest

import (
	"context"
	"testing"
	"time"

	"extra-badges/internal/client"
)

// RunClients exercises client.Repository implementations built by newRepo
// (each call must return an empty store).
func RunClients(t *testing.T, newRepo func(t *testing.T) client.Repository) {
	ctx := context.Background()
	now := time.Date(2026, 1, 2, 3, 4, 5, 6000, time.UTC)

	t.Run("get many on empty store", func(t *testing.T) {
		got, err := newRepo(t).GetMany(ctx, []int64{1})
		if err != nil || len(got) != 0 {
			t.Fatalf("GetMany = %+v, %v", got, err)
		}
	})

	t.Run("touch creates and refreshes", func(t *testing.T) {
		r := newRepo(t)
		if err := r.Touch(ctx, 1, now); err != nil {
			t.Fatal(err)
		}
		_, _ = r.GetMany(ctx, []int64{1}) // warm any cache
		later := now.Add(time.Hour)
		if err := r.Touch(ctx, 1, later); err != nil {
			t.Fatal(err)
		}
		got, err := r.GetMany(ctx, []int64{1, 2})
		if err != nil || len(got) != 1 {
			t.Fatalf("GetMany = %+v, %v", got, err)
		}
		if c := got[1]; c.UserID != 1 || !c.LastSeen.Equal(later) || c.Hidden {
			t.Fatalf("client = %+v", c)
		}
	})

	t.Run("hide and touch again", func(t *testing.T) {
		r := newRepo(t)
		_ = r.Touch(ctx, 1, now)
		_, _ = r.GetMany(ctx, []int64{1})
		if err := r.Hide(ctx, 1); err != nil {
			t.Fatal(err)
		}
		if got, _ := r.GetMany(ctx, []int64{1}); !got[1].Hidden {
			t.Fatalf("not hidden after Hide: %+v", got[1])
		}
		_ = r.Touch(ctx, 1, now)
		if got, _ := r.GetMany(ctx, []int64{1}); got[1].Hidden {
			t.Fatalf("still hidden after Touch: %+v", got[1])
		}
	})

	t.Run("hide unknown is a no-op", func(t *testing.T) {
		r := newRepo(t)
		if err := r.Hide(ctx, 1); err != nil {
			t.Fatal(err)
		}
		if got, _ := r.GetMany(ctx, []int64{1}); len(got) != 0 {
			t.Fatalf("Hide created a record: %+v", got)
		}
	})
}
