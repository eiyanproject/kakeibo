package handlers

import (
	"net/http"
	"strconv"

	"kakeibo/internal/repo"
	"kakeibo/internal/web"
)

func (h *Handlers) AccountCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.FormValue("name")
	institution := r.FormValue("institution")
	accountType := r.FormValue("account_type")
	currency := r.FormValue("currency")
	if currency == "" {
		currency = "JPY"
	}

	_, err := h.Store.CreateAccount(ctx, repo.Account{
		Name: name, Institution: institution, AccountType: accountType, Currency: currency,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handlers) AccountDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	acc, err := h.Store.GetAccount(ctx, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	txns, err := h.Store.ListTransactions(ctx, repo.TransactionFilter{AccountID: &id, Limit: 50})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	web.Render(w, "account_detail.html", map[string]any{
		"Account":      acc,
		"Transactions": txns,
	})
}

func (h *Handlers) AccountArchive(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.Store.ArchiveAccount(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
