package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type BalanceSnapshot struct {
	ID           int64
	AccountID    int64
	SnapshotDate time.Time
	BalanceMinor int64
	CreatedAt    time.Time
}

func (s *Store) AddSnapshot(ctx context.Context, accountID int64, date time.Time, balanceMinor int64) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO balance_snapshots (account_id, snapshot_date, balance_minor)
         VALUES ($1,$2,$3)
         ON CONFLICT (account_id, snapshot_date) DO UPDATE SET balance_minor=EXCLUDED.balance_minor`,
		accountID, date, balanceMinor)
	return err
}

func (s *Store) LatestSnapshot(ctx context.Context, accountID int64, asOf *time.Time) (*BalanceSnapshot, error) {
	q := `SELECT id, account_id, snapshot_date, balance_minor, created_at FROM balance_snapshots WHERE account_id=$1`
	args := []any{accountID}
	if asOf != nil {
		q += ` AND snapshot_date <= $2`
		args = append(args, *asOf)
	}
	q += ` ORDER BY snapshot_date DESC LIMIT 1`
	var b BalanceSnapshot
	err := s.Pool.QueryRow(ctx, q, args...).Scan(&b.ID, &b.AccountID, &b.SnapshotDate, &b.BalanceMinor, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *Store) ListSnapshots(ctx context.Context, accountID int64) ([]BalanceSnapshot, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, account_id, snapshot_date, balance_minor, created_at FROM balance_snapshots WHERE account_id=$1 ORDER BY snapshot_date`,
		accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BalanceSnapshot
	for rows.Next() {
		var b BalanceSnapshot
		if err := rows.Scan(&b.ID, &b.AccountID, &b.SnapshotDate, &b.BalanceMinor, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
