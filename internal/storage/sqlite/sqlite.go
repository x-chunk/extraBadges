// Package sqlite provides a SQLite-backed badge repository.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "modernc.org/sqlite" // registers the "sqlite" driver

	"extra-badges/internal/badge"
)

// migrations[i] upgrades the schema from user_version i to i+1.
// Append only: never edit a migration that has shipped.
var migrations = []string{
	`CREATE TABLE IF NOT EXISTS badges (
	user_id    INTEGER PRIMARY KEY,
	emoji_id   INTEGER NOT NULL,
	title      TEXT    NOT NULL,
	created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	updated_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	)`,
	`ALTER TABLE badges RENAME COLUMN title TO description`,
	`CREATE TABLE clients (
		user_id   INTEGER PRIMARY KEY,
		last_seen TEXT    NOT NULL,
		hidden    INTEGER NOT NULL DEFAULT 0
	)`,
}

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
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return &Repository{db: db}, nil
}

// migrate applies pending migrations, tracking progress in PRAGMA user_version.
func migrate(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	for i := version; i < len(migrations); i++ {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", i+1, err)
		}
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", i+1, err)
		}
		// PRAGMA does not accept bound parameters.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, i+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("bump schema version to %d: %w", i+1, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", i+1, err)
		}
	}
	return nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) Get(ctx context.Context, userID int64) (badge.Badge, error) {
	var b badge.Badge
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, emoji_id, description FROM badges WHERE user_id = ?`, userID,
	).Scan(&b.UserID, &b.EmojiID, &b.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return badge.Badge{}, badge.ErrNotFound
	}
	if err != nil {
		return badge.Badge{}, fmt.Errorf("get badge: %w", err)
	}
	return b, nil
}

func (r *Repository) GetMany(ctx context.Context, ids []int64) (map[int64]badge.Badge, error) {
	out := make(map[int64]badge.Badge, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	rows, err := r.db.QueryContext(ctx,
		`SELECT user_id, emoji_id, description FROM badges WHERE user_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("get badges: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var b badge.Badge
		if err := rows.Scan(&b.UserID, &b.EmojiID, &b.Description); err != nil {
			return nil, fmt.Errorf("scan badge: %w", err)
		}
		out[b.UserID] = b
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get badges: %w", err)
	}
	return out, nil
}

func (r *Repository) List(ctx context.Context) ([]badge.Badge, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT user_id, emoji_id, description FROM badges ORDER BY user_id`)
	if err != nil {
		return nil, fmt.Errorf("list badges: %w", err)
	}
	defer rows.Close()
	out := []badge.Badge{}
	for rows.Next() {
		var b badge.Badge
		if err := rows.Scan(&b.UserID, &b.EmojiID, &b.Description); err != nil {
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
		`INSERT INTO badges (user_id, emoji_id, description) VALUES (?, ?, ?) ON CONFLICT(user_id) DO NOTHING`,
		b.UserID, b.EmojiID, b.Description)
	if err != nil {
		return fmt.Errorf("create badge: %w", err)
	}
	return expectOne(res, badge.ErrAlreadyExists)
}

func (r *Repository) Upsert(ctx context.Context, b badge.Badge) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO badges (user_id, emoji_id, description) VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			emoji_id    = excluded.emoji_id,
			description = excluded.description,
			updated_at  = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`,
		b.UserID, b.EmojiID, b.Description)
	if err != nil {
		return fmt.Errorf("upsert badge: %w", err)
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, b badge.Badge) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE badges SET emoji_id = ?, description = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE user_id = ?`,
		b.EmojiID, b.Description, b.UserID)
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
