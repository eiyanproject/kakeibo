package repo

import (
	"context"
	"strings"
)

type CategoryRule struct {
	ID         int64
	CategoryID int64
	MatchText  string
	Priority   int
}

func (s *Store) ListRules(ctx context.Context) ([]CategoryRule, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, category_id, match_text, priority FROM category_rules ORDER BY priority DESC, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CategoryRule
	for rows.Next() {
		var r CategoryRule
		if err := rows.Scan(&r.ID, &r.CategoryID, &r.MatchText, &r.Priority); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateRule(ctx context.Context, categoryID int64, matchText string, priority int) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO category_rules (category_id, match_text, priority) VALUES ($1,$2,$3) RETURNING id`,
		categoryID, matchText, priority).Scan(&id)
	return id, err
}

func (s *Store) DeleteRule(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM category_rules WHERE id=$1`, id)
	return err
}

// MatchCategory returns the category id for a merchant name based on rules (first match wins, by priority).
func MatchCategory(rules []CategoryRule, merchant string) (int64, bool) {
	lower := strings.ToLower(merchant)
	for _, r := range rules {
		if strings.Contains(lower, strings.ToLower(r.MatchText)) {
			return r.CategoryID, true
		}
	}
	return 0, false
}
