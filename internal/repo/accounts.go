package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Account struct {
	ID                  int64
	Name                string
	Institution         string
	AccountType         string // checking, credit, cash, investment, other
	Currency            string
	OpeningBalanceMinor int64
	OpeningBalanceDate  time.Time
	IsArchived          bool
	CreatedAt           time.Time
}

func (a Account) IsLiability() bool {
	return a.AccountType == "credit"
}

func (s *Store) CreateAccount(ctx context.Context, a Account) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO accounts (name, institution, account_type, currency, opening_balance_minor, opening_balance_date)
         VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		a.Name, a.Institution, a.AccountType, a.Currency, a.OpeningBalanceMinor, a.OpeningBalanceDate,
	).Scan(&id)
	return id, err
}

func (s *Store) ListAccounts(ctx context.Context, includeArchived bool) ([]Account, error) {
	q := `SELECT id, name, institution, account_type, currency, opening_balance_minor, opening_balance_date, is_archived, created_at
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
		if err := rows.Scan(&a.ID, &a.Name, &a.Institution, &a.AccountType, &a.Currency, &a.OpeningBalanceMinor, &a.OpeningBalanceDate, &a.IsArchived, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAccount(ctx context.Context, id int64) (*Account, error) {
	var a Account
	err := s.Pool.QueryRow(ctx,
		`SELECT id, name, institution, account_type, currency, opening_balance_minor, opening_balance_date, is_archived, created_at
         FROM accounts WHERE id=$1`, id,
	).Scan(&a.ID, &a.Name, &a.Institution, &a.AccountType, &a.Currency, &a.OpeningBalanceMinor, &a.OpeningBalanceDate, &a.IsArchived, &a.CreatedAt)
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

// AccountBalance returns opening_balance + transactions as of a given date (nil = all time),
// preferring the latest balance snapshot at/before that date as a base when one exists.
func (s *Store) AccountBalance(ctx context.Context, accountID int64, asOf *time.Time) (int64, error) {
	a, err := s.GetAccount(ctx, accountID)
	if err != nil {
		return 0, err
	}

	snap, err := s.LatestSnapshot(ctx, accountID, asOf)
	if err == nil && snap != nil {
		q := `SELECT COALESCE(SUM(amount_minor),0) FROM transactions WHERE account_id=$1 AND txn_date > $2`
		args := []any{accountID, snap.SnapshotDate}
		if asOf != nil {
			q += ` AND txn_date <= $3`
			args = append(args, *asOf)
		}
		var sumAfter int64
		if err := s.Pool.QueryRow(ctx, q, args...).Scan(&sumAfter); err == nil {
			return snap.BalanceMinor + sumAfter, nil
		}
	}

	q := `SELECT COALESCE(SUM(amount_minor),0) FROM transactions WHERE account_id=$1`
	args := []any{accountID}
	if asOf != nil {
		q += ` AND txn_date <= $2`
		args = append(args, *asOf)
	}
	var sum int64
	if err := s.Pool.QueryRow(ctx, q, args...).Scan(&sum); err != nil {
		return 0, err
	}
	return a.OpeningBalanceMinor + sum, nil
}
