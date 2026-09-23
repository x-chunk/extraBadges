// Package memory provides a thread-safe in-memory badge repository.
package memory

import (
	"context"
	"slices"
	"sync"

	"extra-badges/internal/badge"
)

type Repository struct {
	mu     sync.RWMutex
	badges map[int64]badge.Badge
}

var _ badge.Repository = (*Repository)(nil)

func New(seed ...badge.Badge) *Repository {
	r := &Repository{badges: make(map[int64]badge.Badge, len(seed))}
	for _, b := range seed {
		r.badges[b.UserID] = b
	}
	return r
}

func (r *Repository) Get(_ context.Context, userID int64) (badge.Badge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.badges[userID]
	if !ok {
		return badge.Badge{}, badge.ErrNotFound
	}
	return b, nil
}

func (r *Repository) List(_ context.Context) ([]badge.Badge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]badge.Badge, 0, len(r.badges))
	for _, b := range r.badges {
		out = append(out, b)
	}
	slices.SortFunc(out, func(a, b badge.Badge) int {
		switch {
		case a.UserID < b.UserID:
			return -1
		case a.UserID > b.UserID:
			return 1
		}
		return 0
	})
	return out, nil
}

func (r *Repository) Create(_ context.Context, b badge.Badge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.badges[b.UserID]; ok {
		return badge.ErrAlreadyExists
	}
	r.badges[b.UserID] = b
	return nil
}

func (r *Repository) Upsert(_ context.Context, b badge.Badge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.badges[b.UserID] = b
	return nil
}

func (r *Repository) Update(_ context.Context, b badge.Badge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.badges[b.UserID]; !ok {
		return badge.ErrNotFound
	}
	r.badges[b.UserID] = b
	return nil
}

func (r *Repository) Delete(_ context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.badges[userID]; !ok {
		return badge.ErrNotFound
	}
	delete(r.badges, userID)
	return nil
}
