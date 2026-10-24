package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"extra-badges/internal/client"
)

// Clients is a SQLite-backed client.Repository sharing the badge database.
type Clients struct {
	db *sql.DB
}

var _ client.Repository = (*Clients)(nil)

// Clients returns the client repository backed by the same database.
func (r *Repository) Clients() *Clients {
	return &Clients{db: r.db}
}

func (r *Clients) Touch(ctx context.Context, userID int64, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO clients (user_id, last_seen, hidden) VALUES (?, ?, 0)
		ON CONFLICT(user_id) DO UPDATE SET last_seen = excluded.last_seen, hidden = 0`,
		userID, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("touch client: %w", err)
	}
	return nil
}

func (r *Clients) Hide(ctx context.Context, userID int64) error {
	if _, err := r.db.ExecContext(ctx, `UPDATE clients SET hidden = 1 WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("hide client: %w", err)
	}
	return nil
}

func (r *Clients) GetMany(ctx context.Context, ids []int64) (map[int64]client.Client, error) {
	out := make(map[int64]client.Client, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id, last_seen, hidden FROM clients WHERE user_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("get clients: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			c        client.Client
			lastSeen string
		)
		if err := rows.Scan(&c.UserID, &lastSeen, &c.Hidden); err != nil {
			return nil, fmt.Errorf("scan client: %w", err)
		}
		if c.LastSeen, err = time.Parse(time.RFC3339Nano, lastSeen); err != nil {
			return nil, fmt.Errorf("parse last_seen: %w", err)
		}
		out[c.UserID] = c
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get clients: %w", err)
	}
	return out, nil
}
