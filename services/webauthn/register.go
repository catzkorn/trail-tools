package webauthn

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/catzkorn/trail-tools/store"
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
		h.log.ErrorContext(r.Context(), "failed to get user", slogor.Err(err))
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	options, session, err := h.webauthn.BeginRegistration(user)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to begin registration", slogor.Err(err))
		http.Error(w, "failed to begin registration", http.StatusInternalServerError)
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

func (h *handler) registerFinish(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}
	user, err := h.userRepository.GetWebAuthnUser(r.Context(), []byte(userID))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
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
	cred, err := h.webauthn.FinishRegistration(user, *session, r)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to get session data", slogor.Err(err))
		http.Error(w, "failed to get session data", http.StatusInternalServerError)
		return
	}
	if err := h.userRepository.UpsertWebAuthnCredential(r.Context(), user.WebAuthnID(), cred); err != nil {
		h.log.ErrorContext(r.Context(), "failed to add user credential", slogor.Err(err))
		http.Error(w, "failed to add user credential", http.StatusInternalServerError)
		return
	}
}
