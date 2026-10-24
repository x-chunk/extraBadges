// Package client tracks which users run the exteraless client.
package client

import (
	"context"
	"time"
)

// Client records the last time a user proved they run the client.
type Client struct {
	UserID   int64
	LastSeen time.Time
	Hidden   bool
}

// Visible reports whether the client mark should be shown at now.
func (c Client) Visible(now time.Time, ttl time.Duration) bool {
	return !c.Hidden && now.Sub(c.LastSeen) < ttl
}

// Repository is a storage abstraction for client marks.
type Repository interface {
	// Touch sets last_seen to now and clears hidden, creating the record if needed.
	Touch(ctx context.Context, userID int64, now time.Time) error
	// Hide hides the mark; it is a no-op for unknown users.
	Hide(ctx context.Context, userID int64) error
	// GetMany returns the records that exist among ids; missing ids are omitted.
	GetMany(ctx context.Context, ids []int64) (map[int64]Client, error)
}
