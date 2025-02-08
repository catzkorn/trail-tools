package webauthn

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gitlab.com/greyxor/slogor"
)

func (h *handler) registerBegin(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if len(name) > 256 {
		http.Error(w, "name is too long", http.StatusBadRequest)
		return
	}
	user, err := h.userRepository.CreateWebAuthnUser(r.Context(), name)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to create user", slogor.Err(err))
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}
	options, session, err := h.webauthn.BeginRegistration(user)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to begin registration", slogor.Err(err))
		http.Error(w, "failed to begin registration", http.StatusInternalServerError)
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
		Path:     "/webauthn/register",
		Expires:  session.Expires,
	})
	if err := json.NewEncoder(w).Encode(options); err != nil {
		h.log.ErrorContext(r.Context(), "failed to encode response", slogor.Err(err))
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *handler) registerFinish(w http.ResponseWriter, r *http.Request) {
	webAuthnCookie, err := r.Cookie(webAuthnCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			http.Error(w, "no existing webauthn registration session found", http.StatusForbidden)
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
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		h.log.ErrorContext(r.Context(), "failed to decode session", slogor.Err(err))
		http.Error(w, "failed to decode session", http.StatusInternalServerError)
		return
	}
	dbUser, err := h.userRepository.GetWebAuthnUser(r.Context(), session.UserID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		h.log.ErrorContext(r.Context(), "failed to get user", slogor.Err(err))
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	cred, err := h.webauthn.FinishRegistration(dbUser, session, r)
	if err != nil {
		verErr := new(protocol.Error)
		if errors.As(err, &verErr) {
			h.log.ErrorContext(r.Context(), "failed to verify registration",
				slog.String("details", verErr.Details),
				slog.String("type", verErr.Type),
				slog.String("devinfo", verErr.DevInfo),
			)
		}
		h.log.ErrorContext(r.Context(), "failed to finish registration", slogor.Err(err))
		http.Error(w, "failed to finish registration", http.StatusInternalServerError)
		return
	}
	if err := h.userRepository.UpsertWebAuthnCredential(r.Context(), dbUser.WebAuthnID(), cred); err != nil {
		h.log.ErrorContext(r.Context(), "failed to add user credential", slogor.Err(err))
		http.Error(w, "failed to add user credential", http.StatusInternalServerError)
		return
	}
	// Expire the session cookie
	webAuthnCookie.MaxAge = -1
	http.SetCookie(w, webAuthnCookie)

	// Registration successful, now log in the user
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
