package handlers

import (
	"net/http"
	"strconv"
	"time"

	"kakeibo/internal/repo"
	"kakeibo/internal/web"
)

type accountView struct {
	repo.Account
	BalanceMinor int64
}

func (h *Handlers) AccountsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accounts, err := h.Store.ListAccounts(ctx, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	var views []accountView
	for _, a := range accounts {
		bal, err := h.Store.AccountBalance(ctx, a.ID, &now)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		views = append(views, accountView{Account: a, BalanceMinor: bal})
	}
	web.Render(w, "accounts.html", map[string]any{"Accounts": views})
}

func (h *Handlers) AccountCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.FormValue("name")
	institution := r.FormValue("institution")
	accountType := r.FormValue("account_type")
	currency := r.FormValue("currency")
	if currency == "" {
		currency = "JPY"
	}
	openingF, _ := strconv.ParseFloat(r.FormValue("opening_balance"), 64)
	openingMinor := int64(openingF * 100)

	_, err := h.Store.CreateAccount(ctx, repo.Account{
		Name: name, Institution: institution, AccountType: accountType, Currency: currency,
		OpeningBalanceMinor: openingMinor, OpeningBalanceDate: time.Now(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
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
	now := time.Now()
	bal, err := h.Store.AccountBalance(ctx, id, &now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	snapshots, err := h.Store.ListSnapshots(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	txns, err := h.Store.ListTransactions(ctx, repo.TransactionFilter{AccountID: &id, Limit: 50})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	web.Render(w, "account_detail.html", map[string]any{
		"Account":      acc,
		"BalanceMinor": bal,
		"Snapshots":    snapshots,
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
	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}

func (h *Handlers) AccountAddSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	d, err := time.Parse("2006-01-02", r.FormValue("date"))
	if err != nil {
		http.Error(w, "bad date", http.StatusBadRequest)
		return
	}
	balF, _ := strconv.ParseFloat(r.FormValue("balance"), 64)
	balMinor := int64(balF * 100)
	if err := h.Store.AddSnapshot(r.Context(), id, d, balMinor); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/accounts/"+r.PathValue("id"), http.StatusSeeOther)
}
