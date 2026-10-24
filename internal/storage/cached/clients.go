package cached

import (
	"context"
	"time"

	"github.com/x-chunk/anvil"

	"extra-badges/internal/client"
)

// Clients caches client records per id. As with badges, only hits are cached.
type Clients struct {
	next  client.Repository
	cache *anvil.Cache[int64, client.Client]
}

var _ client.Repository = (*Clients)(nil)

func NewClients(next client.Repository, ttl time.Duration) *Clients {
	return &Clients{next: next, cache: anvil.NewCache[int64, client.Client](ttl)}
}

func (r *Clients) Touch(ctx context.Context, userID int64, now time.Time) error {
	defer r.cache.Invalidate(userID)
	return r.next.Touch(ctx, userID, now)
}

func (r *Clients) Hide(ctx context.Context, userID int64) error {
	defer r.cache.Invalidate(userID)
	return r.next.Hide(ctx, userID)
}

func (r *Clients) GetMany(ctx context.Context, ids []int64) (map[int64]client.Client, error) {
	out := make(map[int64]client.Client, len(ids))
	var misses []int64
	for _, id := range ids {
		if c, ok := r.cache.Get(id); ok {
			out[id] = c
		} else {
			misses = append(misses, id)
		}
	}
	if len(misses) == 0 {
		return out, nil
	}
	fetched, err := r.next.GetMany(ctx, misses)
	if err != nil {
		return nil, err
	}
	for id, c := range fetched {
		r.cache.Set(id, c)
		out[id] = c
	}
	return out, nil
}
