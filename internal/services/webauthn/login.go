package webauthn

import (
	"encoding/json"
	"net/http"

	"gitlab.com/greyxor/slogor"
)

func (h *handler) loginBegin(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	user, err := h.userRepository.GetWebAuthnUser(r.Context(), []byte(userID))
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get user", slogor.Err(err))
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	options, session, err := h.webauthn.BeginLogin(user)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to begin login", slogor.Err(err))
		http.Error(w, "failed to begin login", http.StatusInternalServerError)
		return
	}
	if err := h.userRepository.CreateWebAuthnSession(r.Context(), session); err != nil {
		h.log.ErrorContext(r.Context(), "failed to store session", slogor.Err(err))
		http.Error(w, "failed to store session", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(options); err != nil {
		h.log.ErrorContext(r.Context(), "failed to encode response", slogor.Err(err))
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *handler) loginFinish(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	user, err := h.userRepository.GetWebAuthnUser(r.Context(), []byte(userID))
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get user", slogor.Err(err))
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	session, err := h.userRepository.GetWebAuthnSession(r.Context(), user.WebAuthnID())
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get user session", slogor.Err(err))
		http.Error(w, "failed to get user session", http.StatusInternalServerError)
		return
	}
	cred, err := h.webauthn.FinishLogin(user, *session, r)
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
}
