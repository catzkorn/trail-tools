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

const (
	sessionCookieName = "session"
	logoutPath        = "/logout"
)

type SessionRepository interface {
	GetSession(ctx context.Context, sessionID string) (users.User, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

type authCtx struct{}

func NewAuthnMiddleware(ctx context.Context, log *slog.Logger, repo SessionRepository, next http.Handler) (http.Handler, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie(sessionCookieName)
		if errors.Is(err, http.ErrNoCookie) {
			log.DebugContext(r.Context(), "User had no session cookie, continue unauthenticated", slog.String("path", r.URL.Path))
			next.ServeHTTP(w, r)
			return
		}
		if err != nil {
			log.DebugContext(r.Context(), "Failed to get session cookie, continue unauthenticated", slogor.Err(err), slog.String("path", r.URL.Path))
			next.ServeHTTP(w, r)
			return
		}
		user, err := repo.GetSession(r.Context(), sessionCookie.Value)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				log.DebugContext(r.Context(), "session could not be found, continue unauthenticated")
				expireCookie(w, sessionCookie)
				next.ServeHTTP(w, r)
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

// SetSessionCookie is called by the auth method login handlers on successful login.
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

// RegisterLogoutHandler registers a handler that logs out the user by
// expiring the session cookie and redirecting to the root path.
// extraLogoutHandlers are additional handlers that will be called
// before the redirect.
func RegisterLogoutHandler(logger *slog.Logger, mux *http.ServeMux, sessionRepo SessionRepository, extraLogoutHandlers ...http.Handler) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookieName); err == nil {
			if err := sessionRepo.DeleteSession(r.Context(), c.Value); err != nil && !errors.Is(err, store.ErrNotFound) {
				logger.ErrorContext(r.Context(), "failed to delete session", slogor.Err(err))
			}
			expireCookie(w, c)
		}
		for _, h := range extraLogoutHandlers {
			h.ServeHTTP(w, r)
		}
		http.Redirect(w, r, "/", http.StatusFound)
	})
	mux.Handle(logoutPath, h)
}

func expireCookie(w http.ResponseWriter, oldCookie *http.Cookie) {
	oldCookie.Expires = time.Now().Add(-time.Hour)
	oldCookie.Path = "/"
	oldCookie.Value = ""
	http.SetCookie(w, oldCookie)
}
