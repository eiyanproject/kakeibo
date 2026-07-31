package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"kakeibo/internal/auth"
	"kakeibo/internal/web"
)

func (h *Handlers) Admin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	me := auth.UserFromContext(ctx)

	users, err := h.Store.ListUsers(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accounts, err := h.Store.ListAccounts(ctx, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now()
	var accountViews []accountView
	for _, a := range accounts {
		bal, err := h.Store.AccountBalance(ctx, a.ID, &now)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		accountViews = append(accountViews, accountView{Account: a, BalanceMinor: bal})
	}

	web.Render(w, "admin.html", map[string]any{
		"Me":       me,
		"Users":    users,
		"Accounts": accountViews,
		"Error":    r.URL.Query().Get("error"),
	})
}

func adminError(w http.ResponseWriter, r *http.Request, msg string) {
	http.Redirect(w, r, "/admin?error="+url.QueryEscape(msg), http.StatusSeeOther)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (h *Handlers) AdminUpdateCredentials(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	me := auth.UserFromContext(ctx)

	if !auth.CheckPassword(me.PasswordHash, r.FormValue("current_password")) {
		adminError(w, r, "Current password is incorrect")
		return
	}
	username := r.FormValue("username")
	if username == "" {
		adminError(w, r, "Username is required")
		return
	}
	newPassword := r.FormValue("new_password")
	hash := me.PasswordHash
	if newPassword != "" {
		if newPassword != r.FormValue("confirm_password") {
			adminError(w, r, "New password and confirmation don't match")
			return
		}
		h2, err := auth.HashPassword(newPassword)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hash = h2
	}
	if err := h.Store.UpdateUserCredentials(ctx, me.ID, username, hash); err != nil {
		if isUniqueViolation(err) {
			adminError(w, r, "That username is already taken")
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handlers) AdminCreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" || password != r.FormValue("confirm") {
		adminError(w, r, "Username and matching password are required")
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := h.Store.CreateUser(ctx, username, hash); err != nil {
		if isUniqueViolation(err) {
			adminError(w, r, "That username is already taken")
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *Handlers) AdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	me := auth.UserFromContext(ctx)
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if id == me.ID {
		adminError(w, r, "You can't delete the account you're currently logged in as")
		return
	}
	n, err := h.Store.UserCount(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if n <= 1 {
		adminError(w, r, "At least one user must remain")
		return
	}
	if err := h.Store.DeleteUser(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
