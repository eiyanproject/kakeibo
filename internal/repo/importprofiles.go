package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type ImportProfile struct {
	AccountID    int64
	DateCol      string
	DescCol      string
	AmountCol    string
	DateLayout   string
	InvertAmount bool
	HasHeader    bool
}

func (s *Store) GetImportProfile(ctx context.Context, accountID int64) (*ImportProfile, error) {
	var p ImportProfile
	err := s.Pool.QueryRow(ctx,
		`SELECT account_id, date_col, desc_col, amount_col, date_layout, invert_amount, has_header
         FROM import_profiles WHERE account_id=$1`, accountID,
	).Scan(&p.AccountID, &p.DateCol, &p.DescCol, &p.AmountCol, &p.DateLayout, &p.InvertAmount, &p.HasHeader)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) SaveImportProfile(ctx context.Context, p ImportProfile) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO import_profiles (account_id, date_col, desc_col, amount_col, date_layout, invert_amount, has_header, updated_at)
         VALUES ($1,$2,$3,$4,$5,$6,$7, now())
         ON CONFLICT (account_id) DO UPDATE SET
           date_col=EXCLUDED.date_col, desc_col=EXCLUDED.desc_col, amount_col=EXCLUDED.amount_col,
           date_layout=EXCLUDED.date_layout, invert_amount=EXCLUDED.invert_amount, has_header=EXCLUDED.has_header, updated_at=now()`,
		p.AccountID, p.DateCol, p.DescCol, p.AmountCol, p.DateLayout, p.InvertAmount, p.HasHeader)
	return err
}
