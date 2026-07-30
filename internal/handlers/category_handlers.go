package handlers

import (
	"net/http"
	"strconv"

	"kakeibo/internal/repo"
	"kakeibo/internal/web"
)

func (h *Handlers) CategoriesList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cats, err := h.Store.ListCategories(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rules, err := h.Store.ListRules(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	names := map[int64]string{}
	for _, c := range cats {
		names[c.ID] = c.Name
	}
	web.Render(w, "categories.html", map[string]any{
		"Categories": cats,
		"Rules":      rules,
		"CatNames":   names,
	})
}

func (h *Handlers) CategoryDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cat, err := h.Store.GetCategory(ctx, id)
	if err == repo.ErrNotFound {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	txns, err := h.Store.ListTransactions(ctx, repo.TransactionFilter{CategoryID: &id, Limit: 200})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var totalMinor int64
	for _, t := range txns {
		if t.AmountMinor < 0 {
			totalMinor += -t.AmountMinor
		}
	}
	web.Render(w, "category_detail.html", map[string]any{
		"Category":     cat,
		"TotalMinor":   totalMinor,
		"Transactions": txns,
	})
}

func (h *Handlers) CategoryCreate(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	kind := r.FormValue("kind")
	if kind == "" {
		kind = "expense"
	}
	if _, err := h.Store.CreateCategory(r.Context(), name, kind, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (h *Handlers) RuleCreate(w http.ResponseWriter, r *http.Request) {
	catID, err := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
	if err != nil {
		http.Error(w, "bad category", http.StatusBadRequest)
		return
	}
	priority, _ := strconv.Atoi(r.FormValue("priority"))
	if _, err := h.Store.CreateRule(r.Context(), catID, r.FormValue("match_text"), priority); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (h *Handlers) RuleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.Store.DeleteRule(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}
