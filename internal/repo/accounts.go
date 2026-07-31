package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Account struct {
	ID          int64
	Name        string
	Institution string
	AccountType string // checking, credit, cash, investment, other
	Currency    string
	IsArchived  bool
	CreatedAt   time.Time
}

func (s *Store) CreateAccount(ctx context.Context, a Account) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO accounts (name, institution, account_type, currency)
         VALUES ($1,$2,$3,$4) RETURNING id`,
		a.Name, a.Institution, a.AccountType, a.Currency,
	).Scan(&id)
	return id, err
}

func (s *Store) ListAccounts(ctx context.Context, includeArchived bool) ([]Account, error) {
	q := `SELECT id, name, institution, account_type, currency, is_archived, created_at
          FROM accounts`
	if !includeArchived {
		q += ` WHERE is_archived = false`
	}
	q += ` ORDER BY name`
	rows, err := s.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Institution, &a.AccountType, &a.Currency, &a.IsArchived, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAccount(ctx context.Context, id int64) (*Account, error) {
	var a Account
	err := s.Pool.QueryRow(ctx,
		`SELECT id, name, institution, account_type, currency, is_archived, created_at
         FROM accounts WHERE id=$1`, id,
	).Scan(&a.ID, &a.Name, &a.Institution, &a.AccountType, &a.Currency, &a.IsArchived, &a.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) ArchiveAccount(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE accounts SET is_archived = true WHERE id=$1`, id)
	return err
}
