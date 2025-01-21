package webauthn

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/catzkorn/trail-tools/internal/users"
	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	webAuthnCookieName = "webauthn_session"
)

// UserRepository allows storing and querying of webauthn users and related data.
type UserRepository interface {
	CreateWebAuthnUser(ctx context.Context, name string) (*users.WebAuthnUser, error)
	GetWebAuthnUser(ctx context.Context, webAuthnUserID []byte) (*users.WebAuthnUser, error)
	UpsertWebAuthnCredential(ctx context.Context, webAuthnUserID []byte, credential *webauthn.Credential) error
	CreateWebAuthnSession(ctx context.Context, user *users.WebAuthnUser, expiry time.Time) (string, error)
}

type handler struct {
	log            *slog.Logger
	webauthn       *webauthn.WebAuthn
	userRepository UserRepository
}

// RegisterHandlers registers all the necessary handlers for the web authn service to the mux.
func RegisterHandlers(mux *http.ServeMux, log *slog.Logger, webauthn *webauthn.WebAuthn, userRepository UserRepository) {
	h := handler{
		log:            log,
		webauthn:       webauthn,
		userRepository: userRepository,
	}

	mux.HandleFunc("/webauthn/register/begin", h.registerBegin)
	mux.HandleFunc("/webauthn/register/finish", h.registerFinish)
	mux.HandleFunc("/webauthn/login/begin", h.loginBegin)
	mux.HandleFunc("/webauthn/login/finish", h.loginFinish)
}
