package repo

import (
	"context"
	"time"
)

type ImportBatch struct {
	ID             int64
	AccountID      int64
	SourceFilename string
	RowCount       int
	ImportedAt     time.Time
}

func (s *Store) CreateImportBatch(ctx context.Context, accountID int64, filename string, rowCount int) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO import_batches (account_id, source_filename, row_count) VALUES ($1,$2,$3) RETURNING id`,
		accountID, filename, rowCount).Scan(&id)
	return id, err
}

func (s *Store) ListImportBatches(ctx context.Context, accountID int64) ([]ImportBatch, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, account_id, source_filename, row_count, imported_at FROM import_batches WHERE account_id=$1 ORDER BY imported_at DESC`,
		accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ImportBatch
	for rows.Next() {
		var b ImportBatch
		if err := rows.Scan(&b.ID, &b.AccountID, &b.SourceFilename, &b.RowCount, &b.ImportedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
