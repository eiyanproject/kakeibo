package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

func (s *Store) UserCount(ctx context.Context) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash) VALUES ($1,$2) RETURNING id`,
		username, passwordHash).Scan(&id)
	return id, err
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := s.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE username=$1`,
		username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	var u User
	err := s.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, created_at FROM users WHERE id=$1`,
		id).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
