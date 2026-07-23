package auth

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"kakeibo/internal/repo"
)

const sessionCookieName = "kakeibo_session"
const sessionTTL = 30 * 24 * time.Hour

type ctxKey string

const userCtxKey ctxKey = "user"

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func SessionTTL() time.Duration { return sessionTTL }

func CookieName() string { return sessionCookieName }

func SetSessionCookie(w http.ResponseWriter, sess *repo.Session, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func WithUser(ctx context.Context, u *repo.User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

func UserFromContext(ctx context.Context) *repo.User {
	u, _ := ctx.Value(userCtxKey).(*repo.User)
	return u
}

// RequireAuth wraps handlers requiring a logged-in user, redirecting to /login (or /setup on first run) otherwise.
func RequireAuth(store *repo.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				redirectToLoginOrSetup(w, r, store)
				return
			}
			sess, err := store.GetSession(r.Context(), cookie.Value)
			if err != nil {
				redirectToLoginOrSetup(w, r, store)
				return
			}
			user, err := store.GetUserByID(r.Context(), sess.UserID)
			if err != nil {
				redirectToLoginOrSetup(w, r, store)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
		})
	}
}

func redirectToLoginOrSetup(w http.ResponseWriter, r *http.Request, store *repo.Store) {
	n, err := store.UserCount(r.Context())
	if err == nil && n == 0 {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
