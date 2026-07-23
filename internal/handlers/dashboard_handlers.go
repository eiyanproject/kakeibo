package handlers

import (
	"net/http"
	"time"

	"kakeibo/internal/repo"
	"kakeibo/internal/web"
)

func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	cats, err := h.Store.SpendByCategory(ctx, monthStart, monthEnd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var monthTotal int64
	var labels []string
	var data []int64
	for _, c := range cats {
		monthTotal += c.TotalMinor
		labels = append(labels, c.CategoryName)
		data = append(data, c.TotalMinor)
	}

	netWorth, err := h.Store.NetWorthAsOf(ctx, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	series, err := h.Store.NetWorthSeries(ctx, 12)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var nwLabels []string
	var nwData []int64
	for _, p := range series {
		nwLabels = append(nwLabels, p.Date.Format("2006-01"))
		nwData = append(nwData, p.Total)
	}

	recent, err := h.Store.ListTransactions(ctx, repo.TransactionFilter{Limit: 10})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	web.Render(w, "dashboard.html", map[string]any{
		"MonthTotalMinor":    monthTotal,
		"NetWorthMinor":      netWorth,
		"RecentTransactions": recent,
		"CategoryLabelsJSON": toJS(labels),
		"CategoryDataJSON":   toJS(data),
		"NetWorthLabelsJSON": toJS(nwLabels),
		"NetWorthDataJSON":   toJS(nwData),
	})
}
