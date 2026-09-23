// Package cached wraps a badge.Repository with a TTL read-through cache.
package cached

import (
	"context"
	"time"

	"github.com/x-chunk/anvil"

	"extra-badges/internal/badge"
)

// Repository caches single-badge lookups. Only hits are cached: anvil.Cache
// never evicts expired entries, so caching misses would let arbitrary IDs
// grow the map without bound. List always goes to the underlying store.
type Repository struct {
	next  badge.Repository
	cache *anvil.Cache[int64, badge.Badge]
}

var _ badge.Repository = (*Repository)(nil)

func New(next badge.Repository, ttl time.Duration) *Repository {
	return &Repository{next: next, cache: anvil.NewCache[int64, badge.Badge](ttl)}
}

func (r *Repository) Get(ctx context.Context, userID int64) (badge.Badge, error) {
	if b, ok := r.cache.Get(userID); ok {
		return b, nil
	}
	b, err := r.next.Get(ctx, userID)
	if err != nil {
		return badge.Badge{}, err
	}
	r.cache.Set(userID, b)
	return b, nil
}

func (r *Repository) List(ctx context.Context) ([]badge.Badge, error) {
	return r.next.List(ctx)
}

// Writes invalidate rather than populate the cache, so a concurrent
// reader can never pin a value the store did not end up with.

func (r *Repository) Create(ctx context.Context, b badge.Badge) error {
	defer r.cache.Invalidate(b.UserID)
	return r.next.Create(ctx, b)
}

func (r *Repository) Upsert(ctx context.Context, b badge.Badge) error {
	defer r.cache.Invalidate(b.UserID)
	return r.next.Upsert(ctx, b)
}

func (r *Repository) Update(ctx context.Context, b badge.Badge) error {
	defer r.cache.Invalidate(b.UserID)
	return r.next.Update(ctx, b)
}

func (r *Repository) Delete(ctx context.Context, userID int64) error {
	defer r.cache.Invalidate(userID)
	return r.next.Delete(ctx, userID)
}
