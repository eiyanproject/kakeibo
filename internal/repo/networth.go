package repo

import (
	"context"
	"time"
)

type NetWorthPoint struct {
	Date  time.Time
	Total int64
}

// NetWorthAsOf sums balances of all non-archived accounts, subtracting liability accounts.
func (s *Store) NetWorthAsOf(ctx context.Context, asOf time.Time) (int64, error) {
	accounts, err := s.ListAccounts(ctx, false)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, a := range accounts {
		bal, err := s.AccountBalance(ctx, a.ID, &asOf)
		if err != nil {
			return 0, err
		}
		if a.IsLiability() {
			total -= bal
		} else {
			total += bal
		}
	}
	return total, nil
}

// NetWorthSeries computes net worth at the end of each of the last `months` months.
func (s *Store) NetWorthSeries(ctx context.Context, months int) ([]NetWorthPoint, error) {
	now := time.Now()
	var out []NetWorthPoint
	for i := months - 1; i >= 0; i-- {
		d := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -i, 0)
		end := d.AddDate(0, 1, -1)
		total, err := s.NetWorthAsOf(ctx, end)
		if err != nil {
			return nil, err
		}
		out = append(out, NetWorthPoint{Date: end, Total: total})
	}
	return out, nil
}
