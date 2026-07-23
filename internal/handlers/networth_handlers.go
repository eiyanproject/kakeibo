package handlers

import (
	"net/http"
	"time"

	"kakeibo/internal/web"
)

func (h *Handlers) NetWorth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()
	accounts, err := h.Store.ListAccounts(ctx, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type row struct {
		Name         string
		AccountType  string
		BalanceMinor int64
	}
	var rows []row
	for _, a := range accounts {
		bal, err := h.Store.AccountBalance(ctx, a.ID, &now)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rows = append(rows, row{Name: a.Name, AccountType: a.AccountType, BalanceMinor: bal})
	}
	total, err := h.Store.NetWorthAsOf(ctx, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	series, err := h.Store.NetWorthSeries(ctx, 24)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var labels []string
	var data []int64
	for _, p := range series {
		labels = append(labels, p.Date.Format("2006-01"))
		data = append(data, p.Total)
	}
	web.Render(w, "networth.html", map[string]any{
		"Rows":       rows,
		"Total":      total,
		"LabelsJSON": toJS(labels),
		"DataJSON":   toJS(data),
	})
}
