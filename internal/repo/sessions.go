package repo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Session struct {
	ID        string
	UserID    int64
	ExpiresAt time.Time
}

func newSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Store) CreateSession(ctx context.Context, userID int64, ttl time.Duration) (*Session, error) {
	token, err := newSessionToken()
	if err != nil {
		return nil, err
	}
	expires := time.Now().Add(ttl)
	_, err = s.Pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1,$2,$3)`,
		token, userID, expires)
	if err != nil {
		return nil, err
	}
	return &Session{ID: token, UserID: userID, ExpiresAt: expires}, nil
}

func (s *Store) GetSession(ctx context.Context, id string) (*Session, error) {
	var sess Session
	err := s.Pool.QueryRow(ctx,
		`SELECT id, user_id, expires_at FROM sessions WHERE id=$1 AND expires_at > now()`,
		id).Scan(&sess.ID, &sess.UserID, &sess.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM sessions WHERE id=$1`, id)
	return err
}
