package handlers

import (
	"net/http"
	"strconv"
	"time"

	"kakeibo/internal/repo"
	"kakeibo/internal/web"
)

func (h *Handlers) TransactionsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	f := repo.TransactionFilter{Limit: 200}
	if v := r.URL.Query().Get("account_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.AccountID = &id
		}
	}
	if v := r.URL.Query().Get("category_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.CategoryID = &id
		}
	}
	if v := r.URL.Query().Get("month"); v != "" {
		if t, err := time.Parse("2006-01", v); err == nil {
			from := t
			to := t.AddDate(0, 1, -1)
			f.From = &from
			f.To = &to
		}
	}
	txns, err := h.Store.ListTransactions(ctx, f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	accounts, _ := h.Store.ListAccounts(ctx, false)
	categories, _ := h.Store.ListCategories(ctx)
	web.Render(w, "transactions.html", map[string]any{
		"Transactions": txns,
		"Accounts":     accounts,
		"Categories":   categories,
	})
}

func (h *Handlers) TransactionUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	catID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	if err != nil {
		http.Error(w, "bad category", http.StatusBadRequest)
		return
	}
	if err := h.Store.UpdateTransactionCategory(r.Context(), id, catID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.FormValue("create_rule") == "on" {
		if merchant := r.FormValue("merchant"); merchant != "" {
			if _, err := h.Store.CreateRule(r.Context(), catID, merchant, 0); err == nil {
				if uncategorized, err := h.Store.UncategorizedID(r.Context()); err == nil {
					_, _ = h.Store.ApplyRuleToUncategorized(r.Context(), catID, uncategorized, merchant)
				}
			}
		}
	}

	categories, err := h.Store.ListCategories(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	updated, err := h.Store.GetTransaction(r.Context(), id)
	if err != nil {
		http.Error(w, "transaction not found", http.StatusNotFound)
		return
	}
	web.Render(w, "txn-row", map[string]any{"Txn": updated, "Categories": categories})
}

func (h *Handlers) TransactionUpdateNote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	note := r.FormValue("note")
	if err := h.Store.UpdateTransactionNote(r.Context(), id, note); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
