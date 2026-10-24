package memory

import (
	"context"
	"sync"
	"time"

	"extra-badges/internal/client"
)

// Clients is a thread-safe in-memory client.Repository.
type Clients struct {
	mu      sync.RWMutex
	clients map[int64]client.Client
}

var _ client.Repository = (*Clients)(nil)

func NewClients(seed ...client.Client) *Clients {
	r := &Clients{clients: make(map[int64]client.Client, len(seed))}
	for _, c := range seed {
		r.clients[c.UserID] = c
	}
	return r
}

func (r *Clients) Touch(_ context.Context, userID int64, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[userID] = client.Client{UserID: userID, LastSeen: now}
	return nil
}

func (r *Clients) Hide(_ context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.clients[userID]; ok {
		c.Hidden = true
		r.clients[userID] = c
	}
	return nil
}

func (r *Clients) GetMany(_ context.Context, ids []int64) (map[int64]client.Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[int64]client.Client, len(ids))
	for _, id := range ids {
		if c, ok := r.clients[id]; ok {
			out[id] = c
		}
	}
	return out, nil
}
