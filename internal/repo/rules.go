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

// CreateRule creates a rule matching matchText to categoryID, or if a rule already
// exists for that exact match text, repoints it at categoryID instead of creating a
// duplicate — so re-categorizing the same merchant just updates its remembered rule.
func (s *Store) CreateRule(ctx context.Context, categoryID int64, matchText string, priority int) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO category_rules (category_id, match_text, priority) VALUES ($1,$2,$3)
         ON CONFLICT (match_text) DO UPDATE SET category_id = EXCLUDED.category_id, priority = EXCLUDED.priority
         RETURNING id`,
		categoryID, matchText, priority).Scan(&id)
	return id, err
}

// ApplyRuleToUncategorized re-categorizes every transaction still sitting in
// uncategorizedID whose merchant matches matchText, so a newly remembered rule takes
// effect immediately on past transactions, not just future imports. Returns the number
// of rows updated.
func (s *Store) ApplyRuleToUncategorized(ctx context.Context, categoryID, uncategorizedID int64, matchText string) (int64, error) {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE transactions
         SET category_id = $1
         WHERE category_id = $2
           AND position(lower($3) in lower(merchant_raw)) > 0`,
		categoryID, uncategorizedID, matchText)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
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
