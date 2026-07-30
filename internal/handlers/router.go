package handlers

import (
	"io/fs"
	"net/http"

	"kakeibo/internal/auth"
	"kakeibo/internal/config"
	"kakeibo/internal/repo"
	"kakeibo/internal/web"
)

func NewRouter(store *repo.Store, cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	h := &Handlers{Store: store, Cfg: cfg}

	mux.HandleFunc("GET /setup", h.SetupForm)
	mux.HandleFunc("POST /setup", h.SetupSubmit)
	mux.HandleFunc("GET /login", h.LoginForm)
	mux.HandleFunc("POST /login", h.LoginSubmit)
	mux.HandleFunc("POST /logout", h.Logout)

	staticSub, _ := fs.Sub(web.StaticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	protected := http.NewServeMux()
	protected.HandleFunc("GET /{$}", h.Dashboard)

	protected.HandleFunc("GET /accounts", h.AccountsList)
	protected.HandleFunc("POST /accounts", h.AccountCreate)
	protected.HandleFunc("GET /accounts/{id}", h.AccountDetail)
	protected.HandleFunc("POST /accounts/{id}/archive", h.AccountArchive)
	protected.HandleFunc("POST /accounts/{id}/snapshot", h.AccountAddSnapshot)

	protected.HandleFunc("GET /transactions", h.TransactionsList)
	protected.HandleFunc("POST /transactions/{id}/category", h.TransactionUpdateCategory)
	protected.HandleFunc("POST /transactions/{id}/note", h.TransactionUpdateNote)

	protected.HandleFunc("GET /import", h.ImportForm)
	protected.HandleFunc("POST /import/preview", h.ImportPreview)
	protected.HandleFunc("POST /import/commit", h.ImportCommit)

	protected.HandleFunc("GET /categories", h.CategoriesList)
	protected.HandleFunc("GET /categories/{id}", h.CategoryDetail)
	protected.HandleFunc("POST /categories", h.CategoryCreate)
	protected.HandleFunc("POST /rules", h.RuleCreate)
	protected.HandleFunc("POST /rules/{id}/delete", h.RuleDelete)

	mux.Handle("/", auth.RequireAuth(store)(protected))

	return mux
}
