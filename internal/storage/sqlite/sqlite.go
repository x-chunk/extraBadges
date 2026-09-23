// Package sqlite provides a SQLite-backed badge repository.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" driver

	"extra-badges/internal/badge"
)

const schema = `
CREATE TABLE IF NOT EXISTS badges (
	user_id    INTEGER PRIMARY KEY,
	emoji_id   INTEGER NOT NULL,
	title      TEXT    NOT NULL,
	created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	updated_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);`

type Repository struct {
	db *sql.DB
}

var _ badge.Repository = (*Repository)(nil)

// Open opens (or creates) the database at path and applies the schema.
// Use ":memory:" for an ephemeral database.
func Open(ctx context.Context, path string) (*Repository, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite allows a single writer; one connection avoids SQLITE_BUSY
	// and keeps ":memory:" databases shared across calls.
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) Get(ctx context.Context, userID int64) (badge.Badge, error) {
	var b badge.Badge
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, emoji_id, title FROM badges WHERE user_id = ?`, userID,
	).Scan(&b.UserID, &b.EmojiID, &b.Title)
	if errors.Is(err, sql.ErrNoRows) {
		return badge.Badge{}, badge.ErrNotFound
	}
	if err != nil {
		return badge.Badge{}, fmt.Errorf("get badge: %w", err)
	}
	return b, nil
}

func (r *Repository) List(ctx context.Context) ([]badge.Badge, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id, emoji_id, title FROM badges ORDER BY user_id`)
	if err != nil {
		return nil, fmt.Errorf("list badges: %w", err)
	}
	defer rows.Close()
	out := []badge.Badge{}
	for rows.Next() {
		var b badge.Badge
		if err := rows.Scan(&b.UserID, &b.EmojiID, &b.Title); err != nil {
			return nil, fmt.Errorf("scan badge: %w", err)
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list badges: %w", err)
	}
	return out, nil
}

func (r *Repository) Create(ctx context.Context, b badge.Badge) error {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO badges (user_id, emoji_id, title) VALUES (?, ?, ?) ON CONFLICT(user_id) DO NOTHING`,
		b.UserID, b.EmojiID, b.Title)
	if err != nil {
		return fmt.Errorf("create badge: %w", err)
	}
	return expectOne(res, badge.ErrAlreadyExists)
}

func (r *Repository) Upsert(ctx context.Context, b badge.Badge) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO badges (user_id, emoji_id, title) VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			emoji_id   = excluded.emoji_id,
			title      = excluded.title,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`,
		b.UserID, b.EmojiID, b.Title)
	if err != nil {
		return fmt.Errorf("upsert badge: %w", err)
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, b badge.Badge) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE badges SET emoji_id = ?, title = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE user_id = ?`,
		b.EmojiID, b.Title, b.UserID)
	if err != nil {
		return fmt.Errorf("update badge: %w", err)
	}
	return expectOne(res, badge.ErrNotFound)
}

func (r *Repository) Delete(ctx context.Context, userID int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM badges WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete badge: %w", err)
	}
	return expectOne(res, badge.ErrNotFound)
}

func expectOne(res sql.Result, errIfNone error) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return errIfNone
	}
	return nil
}
