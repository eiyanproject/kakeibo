package handlers

import (
	"net/http"

	"kakeibo/internal/auth"
	"kakeibo/internal/web"
)

func (h *Handlers) SetupForm(w http.ResponseWriter, r *http.Request) {
	n, err := h.Store.UserCount(r.Context())
	if err == nil && n > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	web.Render(w, "setup.html", nil)
}

func (h *Handlers) SetupSubmit(w http.ResponseWriter, r *http.Request) {
	n, err := h.Store.UserCount(r.Context())
	if err == nil && n > 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	confirm := r.FormValue("confirm")
	if username == "" || password == "" || password != confirm {
		web.Render(w, "setup.html", map[string]any{"Error": "Username and password are required, and must match confirmation."})
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := h.Store.CreateUser(r.Context(), username, hash); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handlers) LoginForm(w http.ResponseWriter, r *http.Request) {
	n, err := h.Store.UserCount(r.Context())
	if err == nil && n == 0 {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	web.Render(w, "login.html", nil)
}

func (h *Handlers) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	user, err := h.Store.GetUserByUsername(r.Context(), username)
	if err != nil || !auth.CheckPassword(user.PasswordHash, password) {
		web.Render(w, "login.html", map[string]any{"Error": "Invalid username or password"})
		return
	}
	sess, err := h.Store.CreateSession(r.Context(), user.ID, auth.SessionTTL())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	auth.SetSessionCookie(w, sess, h.Cfg.SessionSecure)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.CookieName()); err == nil {
		_ = h.Store.DeleteSession(r.Context(), cookie.Value)
	}
	auth.ClearSessionCookie(w, h.Cfg.SessionSecure)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
