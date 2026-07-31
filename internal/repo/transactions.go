package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Transaction struct {
	ID                 int64
	AccountID          int64
	TxnDate            time.Time
	MerchantRaw        string
	MerchantNormalized string
	AmountMinor        int64
	CategoryID         *int64
	CategoryName       string
	Note               string
	ImportBatchID      *int64
	DedupHash          string
	CreatedAt          time.Time
}

type TransactionFilter struct {
	AccountID  *int64
	CategoryID *int64
	From       *time.Time
	To         *time.Time
	Limit      int
	Offset     int
}

// InsertTransaction inserts a transaction, skipping silently on a (account_id, dedup_hash) conflict.
// The bool return reports whether a new row was actually inserted.
func (s *Store) InsertTransaction(ctx context.Context, t Transaction) (int64, bool, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO transactions (account_id, txn_date, merchant_raw, merchant_normalized, amount_minor, category_id, note, import_batch_id, dedup_hash)
         VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
         ON CONFLICT (account_id, dedup_hash) DO NOTHING
         RETURNING id`,
		t.AccountID, t.TxnDate, t.MerchantRaw, t.MerchantNormalized, t.AmountMinor, t.CategoryID, t.Note, t.ImportBatchID, t.DedupHash,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func (s *Store) ListTransactions(ctx context.Context, f TransactionFilter) ([]Transaction, error) {
	base := `SELECT t.id, t.account_id, t.txn_date, t.merchant_raw, t.merchant_normalized, t.amount_minor, t.category_id, COALESCE(c.name,'Uncategorized'), t.note, t.import_batch_id, t.dedup_hash, t.created_at
             FROM transactions t LEFT JOIN categories c ON c.id = t.category_id WHERE 1=1`
	var args []any
	if f.AccountID != nil {
		args = append(args, *f.AccountID)
		base += fmt.Sprintf(" AND t.account_id = $%d", len(args))
	}
	if f.CategoryID != nil {
		args = append(args, *f.CategoryID)
		base += fmt.Sprintf(" AND t.category_id = $%d", len(args))
	}
	if f.From != nil {
		args = append(args, *f.From)
		base += fmt.Sprintf(" AND t.txn_date >= $%d", len(args))
	}
	if f.To != nil {
		args = append(args, *f.To)
		base += fmt.Sprintf(" AND t.txn_date <= $%d", len(args))
	}
	base += " ORDER BY t.txn_date DESC, t.id DESC"
	if f.Limit > 0 {
		args = append(args, f.Limit)
		base += fmt.Sprintf(" LIMIT $%d", len(args))
	}
	if f.Offset > 0 {
		args = append(args, f.Offset)
		base += fmt.Sprintf(" OFFSET $%d", len(args))
	}
	rows, err := s.Pool.Query(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.TxnDate, &t.MerchantRaw, &t.MerchantNormalized, &t.AmountMinor, &t.CategoryID, &t.CategoryName, &t.Note, &t.ImportBatchID, &t.DedupHash, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTransaction(ctx context.Context, id int64) (*Transaction, error) {
	var t Transaction
	err := s.Pool.QueryRow(ctx,
		`SELECT t.id, t.account_id, t.txn_date, t.merchant_raw, t.merchant_normalized, t.amount_minor, t.category_id, COALESCE(c.name,'Uncategorized'), t.note, t.import_batch_id, t.dedup_hash, t.created_at
         FROM transactions t LEFT JOIN categories c ON c.id = t.category_id WHERE t.id=$1`, id,
	).Scan(&t.ID, &t.AccountID, &t.TxnDate, &t.MerchantRaw, &t.MerchantNormalized, &t.AmountMinor, &t.CategoryID, &t.CategoryName, &t.Note, &t.ImportBatchID, &t.DedupHash, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) UpdateTransactionCategory(ctx context.Context, id, categoryID int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE transactions SET category_id=$1 WHERE id=$2`, categoryID, id)
	return err
}

func (s *Store) UpdateTransactionNote(ctx context.Context, id int64, note string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE transactions SET note=$1 WHERE id=$2`, note, id)
	return err
}

// LatestTransactionMonth returns the first-of-month date for the most recent transaction
// that actually counts as spend (amount_minor < 0), so callers can fall back to it when
// the current calendar month has no data. Restricting to spend rows matters: a stray
// non-spend row (e.g. left over from before a sign-convention fix) in a later month must
// not shadow real spend data sitting in an earlier month.
func (s *Store) LatestTransactionMonth(ctx context.Context, accountID *int64) (time.Time, bool, error) {
	q := `SELECT date_trunc('month', max(txn_date))::date FROM transactions WHERE amount_minor < 0`
	args := []any{}
	if accountID != nil {
		args = append(args, *accountID)
		q += fmt.Sprintf(" AND account_id = $%d", len(args))
	}
	var t *time.Time
	err := s.Pool.QueryRow(ctx, q, args...).Scan(&t)
	if err != nil {
		return time.Time{}, false, err
	}
	if t == nil {
		return time.Time{}, false, nil
	}
	return *t, true, nil
}

type MonthTotal struct {
	Month      time.Time
	TotalMinor int64
}

// SpendByMonth sums outgoing (negative) transactions in [from, to], grouped by calendar month.
// Months with no spending are simply absent from the result — callers that need a fixed-length
// series (e.g. every month in a 12-month window) should fill in zeros themselves.
// accountID narrows to a single account; nil aggregates across all accounts.
func (s *Store) SpendByMonth(ctx context.Context, from, to time.Time, accountID *int64) ([]MonthTotal, error) {
	q := `SELECT date_trunc('month', txn_date)::date as m, SUM(-amount_minor) as total
         FROM transactions
         WHERE txn_date BETWEEN $1 AND $2 AND amount_minor < 0`
	args := []any{from, to}
	if accountID != nil {
		args = append(args, *accountID)
		q += fmt.Sprintf(" AND account_id = $%d", len(args))
	}
	q += ` GROUP BY m ORDER BY m`
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MonthTotal
	for rows.Next() {
		var m MonthTotal
		if err := rows.Scan(&m.Month, &m.TotalMinor); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

type CategoryTotal struct {
	CategoryID   int64
	CategoryName string
	TotalMinor   int64
}

// SpendByCategory sums outgoing (negative) transactions in [from, to] as positive spend totals per category.
// Transactions with no category assigned are grouped under uncategorizedID so every row has a real,
// linkable category id. accountID narrows to a single account; nil aggregates across all accounts.
func (s *Store) SpendByCategory(ctx context.Context, from, to time.Time, uncategorizedID int64, accountID *int64) ([]CategoryTotal, error) {
	args := []any{from, to, uncategorizedID}
	q := `SELECT COALESCE(c.id, $3) as cat_id, COALESCE(c.name,'Uncategorized') as name, SUM(-t.amount_minor) as total
         FROM transactions t LEFT JOIN categories c ON c.id=t.category_id
         WHERE t.txn_date BETWEEN $1 AND $2 AND t.amount_minor < 0`
	if accountID != nil {
		args = append(args, *accountID)
		q += fmt.Sprintf(" AND t.account_id = $%d", len(args))
	}
	q += ` GROUP BY cat_id, name ORDER BY total DESC`
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CategoryTotal
	for rows.Next() {
		var c CategoryTotal
		if err := rows.Scan(&c.CategoryID, &c.CategoryName, &c.TotalMinor); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
