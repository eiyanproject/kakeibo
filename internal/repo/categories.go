package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type Category struct {
	ID       int64
	Name     string
	ParentID *int64
	Kind     string
}

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name, parent_id, kind FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.Kind); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateCategory(ctx context.Context, name, kind string, parentID *int64) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO categories (name, kind, parent_id) VALUES ($1,$2,$3) RETURNING id`,
		name, kind, parentID).Scan(&id)
	return id, err
}

func (s *Store) UpdateCategory(ctx context.Context, id int64, name, kind string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE categories SET name=$1, kind=$2 WHERE id=$3`, name, kind, id)
	return err
}

func (s *Store) DeleteCategory(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM categories WHERE id=$1`, id)
	return err
}

func (s *Store) GetCategory(ctx context.Context, id int64) (*Category, error) {
	var c Category
	err := s.Pool.QueryRow(ctx,
		`SELECT id, name, parent_id, kind FROM categories WHERE id=$1`, id,
	).Scan(&c.ID, &c.Name, &c.ParentID, &c.Kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) UncategorizedID(ctx context.Context) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx, `SELECT id FROM categories WHERE name='Uncategorized' LIMIT 1`).Scan(&id)
	return id, err
}
