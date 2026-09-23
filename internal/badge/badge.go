// Package badge contains the domain model and business rules for badges.
package badge

import (
	"context"
	"errors"
	"strings"
)

const DefaultTitle = "This account has a badge provided by the extraBadges."

var (
	ErrNotFound      = errors.New("badge not found")
	ErrAlreadyExists = errors.New("badge already exists")
	ErrInvalid       = errors.New("invalid badge")
)

// Badge links a user to a custom emoji.
type Badge struct {
	UserID  int64  `json:"user_id"`
	EmojiID int64  `json:"emoji_id"`
	Title   string `json:"title"`
}

// Validate checks the badge and fills defaults.
func (b *Badge) Validate() error {
	if b.UserID <= 0 {
		return errors.Join(ErrInvalid, errors.New("user_id must be positive"))
	}
	if b.EmojiID <= 0 {
		return errors.Join(ErrInvalid, errors.New("emoji_id must be positive"))
	}
	b.Title = strings.TrimSpace(b.Title)
	if b.Title == "" {
		b.Title = DefaultTitle
	}
	return nil
}

// Repository is a storage abstraction for badges.
type Repository interface {
	Get(ctx context.Context, userID int64) (Badge, error)
	List(ctx context.Context) ([]Badge, error)
	Create(ctx context.Context, b Badge) error
	Upsert(ctx context.Context, b Badge) error
	Delete(ctx context.Context, userID int64) error
}

// Service implements badge use cases on top of a Repository.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, userID int64) (Badge, error) {
	if userID <= 0 {
		return Badge{}, errors.Join(ErrInvalid, errors.New("user_id must be positive"))
	}
	return s.repo.Get(ctx, userID)
}

func (s *Service) List(ctx context.Context) ([]Badge, error) {
	return s.repo.List(ctx)
}

func (s *Service) Create(ctx context.Context, b Badge) (Badge, error) {
	if err := b.Validate(); err != nil {
		return Badge{}, err
	}
	if err := s.repo.Create(ctx, b); err != nil {
		return Badge{}, err
	}
	return b, nil
}

func (s *Service) Put(ctx context.Context, b Badge) (Badge, error) {
	if err := b.Validate(); err != nil {
		return Badge{}, err
	}
	if err := s.repo.Upsert(ctx, b); err != nil {
		return Badge{}, err
	}
	return b, nil
}

func (s *Service) Delete(ctx context.Context, userID int64) error {
	return s.repo.Delete(ctx, userID)
}
