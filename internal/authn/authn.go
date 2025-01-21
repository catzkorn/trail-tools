package authn

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users"
	"gitlab.com/greyxor/slogor"
)

const sessionCookieName = "session"

type SessionRepository interface {
	GetSession(ctx context.Context, sessionID string) (users.User, error)
}

type authCtx struct{}

func NewAuthnMiddleware(ctx context.Context, log *slog.Logger, repo SessionRepository, next http.Handler) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie(sessionCookieName)
		if errors.Is(err, http.ErrNoCookie) {
			log.DebugContext(r.Context(), "User had no token, continue unauthenticated", slogor.Err(err), slog.String("path", r.URL.Path))
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			log.ErrorContext(r.Context(), "failed to get session cookie", slogor.Err(err))
			http.Error(w, "failed to get session cookie", http.StatusInternalServerError)
			return
		}
		user, err := repo.GetSession(r.Context(), sessionCookie.Value)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				log.ErrorContext(r.Context(), "session could not be found", slogor.Err(err))
				expireCookie(w, sessionCookie)
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			log.ErrorContext(r.Context(), "failed to get session", slogor.Err(err))
			http.Error(w, "failed to get session", http.StatusInternalServerError)
			return
		}
		newCtx := context.WithValue(r.Context(), authCtx{}, user)
		next.ServeHTTP(w, r.WithContext(newCtx))
	}), nil
}

// GetUser gets the authenticated user, if there is one. It returns
// (nil, false) if the current request is unauthenticated.
func GetUser(ctx context.Context) (users.User, bool) {
	user, ok := ctx.Value(authCtx{}).(users.User)
	return user, ok
}

func SetSessionCookie(w http.ResponseWriter, sessionID string, expiry time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		MaxAge:   int(time.Until(expiry).Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	})
}

func expireCookie(w http.ResponseWriter, oldCookie *http.Cookie) {
	oldCookie.Expires = time.Now().Add(-time.Hour)
	http.SetCookie(w, oldCookie)
}
