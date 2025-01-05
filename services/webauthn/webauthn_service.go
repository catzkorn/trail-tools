package webauthn

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
)

// UserRepository allows storing and querying of webauthn users and related data.
type UserRepository interface {
	CreateWebAuthnUser(ctx context.Context, name string) (webauthn.User, error)
	GetWebAuthnUser(ctx context.Context, webAuthnUserID []byte) (webauthn.User, error)
	CreateWebAuthnSession(ctx context.Context, session *webauthn.SessionData) error
	GetWebAuthnSession(ctx context.Context, webAuthnUserID []byte) (*webauthn.SessionData, error)
	UpsertWebAuthnCredential(ctx context.Context, webAuthnUserID []byte, credential *webauthn.Credential) error
}

type handler struct {
	log            *slog.Logger
	webauthn       *webauthn.WebAuthn
	userRepository UserRepository
}

// Register registers all the necessary handlers for the webauthn service to the mux.
func Register(mux *http.ServeMux, log *slog.Logger, webauthn *webauthn.WebAuthn, userRepository UserRepository) {
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
