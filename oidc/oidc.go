package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/catzkorn/trail-tools/users"
	"github.com/coreos/go-oidc/v3/oidc"
	"gitlab.com/greyxor/slogor"
	"golang.org/x/oauth2"
)

const (
	cookieState   = "oidc-state"
	cookieNonce   = "oidc-nonce"
	cookieIDToken = "oidc-id-token"
	loginPath     = "/oidc/login"
	logoutPath    = "/oidc/logout"
	callbackPath  = "/oidc/callback"
)

type UserRepository interface {
	GetOIDCUser(ctx context.Context, oidcSubject string) (*users.OIDCUser, error)
}

type handler struct {
	log            *slog.Logger
	oauth2Config   oauth2.Config
	verifier       *oidc.IDTokenVerifier
	userRepository UserRepository
}

func RegisterHandlers(
	ctx context.Context,
	logger *slog.Logger,
	baseURL string,
	clientID string,
	clientSecret string,
	issuerURL string,
	userRepository UserRepository,
	mux *http.ServeMux,
) error {
	switch {
	case logger == nil:
		return errors.New("logger is required")
	case baseURL == "":
		return errors.New("base URL is required")
	case clientID == "":
		return errors.New("client ID is required")
	case clientSecret == "":
		return errors.New("client secret is required")
	case issuerURL == "":
		return errors.New("issuer URL is required")
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return errors.New("failed to parse base URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("base URL must have a scheme of http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("base URL must have a host")
	}
	redirectURL := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   callbackPath,
	}
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return fmt.Errorf("failed to create OIDC provider: %w", err)
	}
	h := &handler{
		log: logger,
		oauth2Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL.String(),
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier:       provider.Verifier(&oidc.Config{ClientID: clientID}),
		userRepository: userRepository,
	}
	mux.Handle(loginPath, http.HandlerFunc(h.Login))
	mux.Handle(logoutPath, http.HandlerFunc(h.Logout))
	mux.Handle(callbackPath, http.HandlerFunc(h.Callback))
	return nil
}

func (h *handler) Login(w http.ResponseWriter, r *http.Request) {
	state, err := randBase64(16)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to generate state", slogor.Err(err))
		http.Error(w, "login failed", http.StatusInternalServerError)
		return
	}
	nonce, err := randBase64(16)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to generate nonce", slogor.Err(err))
		http.Error(w, "login failed", http.StatusInternalServerError)
		return
	}
	setCallbackCookie(w, r, cookieState, state)
	setCallbackCookie(w, r, cookieNonce, nonce)
	http.Redirect(w, r, h.oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
}

func (h *handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieIDToken); err == nil {
		c.Expires = time.Now().Add(-time.Hour)
		c.Path = "/"
		http.SetCookie(w, c)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func (h *handler) Callback(w http.ResponseWriter, r *http.Request) {
	state, err := r.Cookie(cookieState)
	if err != nil {
		h.log.ErrorContext(r.Context(), "missing state cookie", slogor.Err(err))
		http.Error(w, "missing state cookie", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != state.Value {
		h.log.ErrorContext(r.Context(), "invalid state", slogor.Err(err))
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	oauth2Token, err := h.oauth2Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to exchange token", slogor.Err(err))
		http.Error(w, "failed to exchange with OIDC issuer", http.StatusInternalServerError)
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		h.log.ErrorContext(r.Context(), "missing ID token in OIDC issuer response", slogor.Err(err))
		http.Error(w, "missing ID token in OIDC issuer response", http.StatusInternalServerError)
		return
	}
	idToken, err := h.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		h.log.ErrorContext(r.Context(), "failed to verify ID token", slogor.Err(err))
		http.Error(w, "failed to verify ID token", http.StatusInternalServerError)
		return
	}
	nonce, err := r.Cookie(cookieNonce)
	if err != nil {
		http.Error(w, "nonce not found", http.StatusBadRequest)
		return
	}
	if idToken.Nonce != nonce.Value {
		http.Error(w, "nonce did not match", http.StatusBadRequest)
		return
	}

	// Create the user in the DB
	if _, err := h.userRepository.GetOIDCUser(r.Context(), idToken.Subject); err != nil {
		h.log.ErrorContext(r.Context(), "failed to get OIDC user", slogor.Err(err))
		http.Error(w, "failed to get OIDC user", http.StatusInternalServerError)
		return
	}

	var userInfo oidc.UserInfo
	if err := idToken.Claims(&userInfo); err != nil {
		h.log.ErrorContext(r.Context(), "failed to get claims from ID token", slogor.Err(err))
		http.Error(w, "failed to get claims from ID token", http.StatusInternalServerError)
		return
	}
	h.log.InfoContext(r.Context(), "User authenticated", slog.String("subject", userInfo.Subject))

	cookieId := &http.Cookie{
		Name:     cookieIDToken,
		Value:    rawIDToken,
		MaxAge:   int(time.Until(idToken.Expiry).Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}
	http.SetCookie(w, cookieId)

	if c, err := r.Cookie(cookieState); err == nil {
		expireCallbackCookie(w, c)
	}
	if c, err := r.Cookie(cookieNonce); err == nil {
		expireCallbackCookie(w, c)
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func randBase64(size int) (string, error) {
	b := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setCallbackCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
	}
	http.SetCookie(w, c)
}

func expireCallbackCookie(w http.ResponseWriter, oldCookie *http.Cookie) {
	oldCookie.Expires = time.Now().Add(-time.Hour)
	http.SetCookie(w, oldCookie)
}
