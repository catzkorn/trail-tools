package webauthn

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/users"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gitlab.com/greyxor/slogor"
)

func (h *handler) loginBegin(w http.ResponseWriter, r *http.Request) {
	options, session, err := h.webauthn.BeginDiscoverableLogin()
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to begin login", slogor.Err(err))
		http.Error(w, "failed to begin login", http.StatusInternalServerError)
		return
	}
	cookieVal, err := json.Marshal(session)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to encode session", slogor.Err(err))
		http.Error(w, "failed to encode session", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: webAuthnCookieName,
		// Encode with base64 to avoid issues with quotes in the cookie value
		Value:    base64.RawURLEncoding.EncodeToString(cookieVal),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/webauthn/login",
		Expires:  session.Expires,
	})
	if err := json.NewEncoder(w).Encode(options); err != nil {
		h.log.ErrorContext(r.Context(), "failed to encode response", slogor.Err(err))
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *handler) loginFinish(w http.ResponseWriter, r *http.Request) {
	parsedResponse, err := protocol.ParseCredentialRequestResponse(r)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to parse credential response", slogor.Err(err))
		http.Error(w, "failed to parse credential response", http.StatusBadRequest)
		return
	}
	webAuthnCookie, err := r.Cookie(webAuthnCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			http.Error(w, "no existing webauthn login session found", http.StatusForbidden)
			return
		}
		h.log.ErrorContext(r.Context(), "failed to get session cookie", slogor.Err(err))
		http.Error(w, "failed to get session cookie", http.StatusInternalServerError)
		return
	}
	sessionData, err := base64.RawURLEncoding.DecodeString(webAuthnCookie.Value)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to decode session", slogor.Err(err))
		http.Error(w, "failed to decode session", http.StatusInternalServerError)
		return
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(sessionData, &session); err != nil {
		h.log.ErrorContext(r.Context(), "failed to decode session", slogor.Err(err))
		http.Error(w, "failed to decode session", http.StatusInternalServerError)
		return
	}
	var dbUser *users.WebAuthnUser
	lookupUser := func(rawID, webAuthnUserID []byte) (webauthn.User, error) {
		dbUser, err = h.userRepository.GetWebAuthnUser(r.Context(), []byte(webAuthnUserID))
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
		return dbUser, nil
	}
	user, cred, err := h.webauthn.ValidatePasskeyLogin(lookupUser, session, parsedResponse)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get session data", slogor.Err(err))
		http.Error(w, "failed to get session data", http.StatusInternalServerError)
		return
	}
	if cred.Authenticator.CloneWarning {
		h.log.ErrorContext(r.Context(), "credential reuse detected", slogor.Err(err))
		http.Error(w, "WebAuthn credential reuse is not permitted by this service", http.StatusForbidden)
		return
	}
	if err := h.userRepository.UpsertWebAuthnCredential(r.Context(), user.WebAuthnID(), cred); err != nil {
		h.log.ErrorContext(r.Context(), "failed to add user credential", slogor.Err(err))
		http.Error(w, "failed to add user credential", http.StatusInternalServerError)
		return
	}
	// Expire the webauthn session cookie
	webAuthnCookie.MaxAge = -1
	http.SetCookie(w, webAuthnCookie)

	// Log the user in immediately after passkey registration
	// 7 days sounds like enough time
	sessionExpiry := time.Now().AddDate(0, 0, 7)
	sessionID, err := h.userRepository.CreateWebAuthnSession(r.Context(), dbUser, sessionExpiry)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to create session", slogor.Err(err))
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	authn.SetSessionCookie(w, sessionID, sessionExpiry)
	http.Redirect(w, r, "/", http.StatusFound)
}
