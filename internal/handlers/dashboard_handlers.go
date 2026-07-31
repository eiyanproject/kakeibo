package handlers

import (
	"net/http"
	"strconv"
	"time"

	"kakeibo/internal/repo"
	"kakeibo/internal/web"
)

func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	uncategorized, err := h.Store.UncategorizedID(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var accountID *int64
	if v := r.URL.Query().Get("account_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			accountID = &id
		}
	}
	accounts, err := h.Store.ListAccounts(ctx, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	explicitMonth := false
	if v := r.URL.Query().Get("month"); v != "" {
		if t, err := time.Parse("2006-01", v); err == nil {
			monthStart = t
			explicitMonth = true
		}
	}

	cats, err := h.Store.SpendByCategory(ctx, monthStart, monthStart.AddDate(0, 1, -1), uncategorized, accountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(cats) == 0 && !explicitMonth {
		// No spending in the current calendar month (e.g. a historical statement was just
		// imported) — fall back to the most recent month that actually has transactions.
		// Only applies when the user hasn't explicitly picked a month to view.
		if latest, ok, err := h.Store.LatestTransactionMonth(ctx, accountID); err == nil && ok {
			monthStart = latest
			cats, err = h.Store.SpendByCategory(ctx, monthStart, monthStart.AddDate(0, 1, -1), uncategorized, accountID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	var monthTotal int64
	var labels []string
	var data []int64
	var ids []int64
	for _, c := range cats {
		monthTotal += c.TotalMinor
		labels = append(labels, c.CategoryName)
		data = append(data, c.TotalMinor)
		ids = append(ids, c.CategoryID)
	}

	recent, err := h.Store.ListTransactions(ctx, repo.TransactionFilter{Limit: 10, AccountID: accountID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Trailing 12 calendar months ending this month, independent of the month picker above.
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	yearStart := currentMonth.AddDate(0, -11, 0)
	monthTotals, err := h.Store.SpendByMonth(ctx, yearStart, currentMonth.AddDate(0, 1, -1), accountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	byMonth := map[string]int64{}
	for _, m := range monthTotals {
		byMonth[m.Month.Format("2006-01")] = m.TotalMinor
	}
	var yearLabels []string
	var yearData []int64
	var yearTotal int64
	for cursor := yearStart; !cursor.After(currentMonth); cursor = cursor.AddDate(0, 1, 0) {
		v := byMonth[cursor.Format("2006-01")]
		yearLabels = append(yearLabels, cursor.Format("Jan 2006"))
		yearData = append(yearData, v)
		yearTotal += v
	}

	var selectedAccountID int64
	if accountID != nil {
		selectedAccountID = *accountID
	}

	web.Render(w, "dashboard.html", map[string]any{
		"MonthTotalMinor":    monthTotal,
		"MonthValue":         monthStart.Format("2006-01"),
		"PeriodLabel":        monthStart.Format("January 2006"),
		"Accounts":           accounts,
		"SelectedAccountID":  selectedAccountID,
		"RecentTransactions": recent,
		"CategoryLabelsJSON": toJS(labels),
		"CategoryDataJSON":   toJS(data),
		"CategoryIDsJSON":    toJS(ids),
		"YearTotalMinor":     yearTotal,
		"YearLabelsJSON":     toJS(yearLabels),
		"YearDataJSON":       toJS(yearData),
	})
}
