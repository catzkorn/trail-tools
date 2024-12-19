package oidc

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"gitlab.com/greyxor/slogor"
)

type UserInfo struct {
	Subject string
}

type authKey struct{}

func GetAuthDetails(ctx context.Context) *UserInfo {
	details, _ := ctx.Value(authKey{}).(*UserInfo)
	return details
}

func NewAuthnMiddleware(ctx context.Context, logger *slog.Logger, issuerURL string, clientID string, next http.Handler) (http.Handler, error) {
	switch {
	case logger == nil:
		return nil, errors.New("logger is required")
	case issuerURL == "":
		return nil, errors.New("issuer URL is required")
	case clientID == "":
		return nil, errors.New("client ID is required")
	case next == nil:
		return nil, errors.New("wrapped HTTP handler is required")
	}
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, errors.New("failed to create OIDC provider")
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: clientID})

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		idCookie, err := req.Cookie(cookieIDToken)
		if err != nil {
			logger.Debug("User had no token, continue unauthenticated", slogor.Err(err))
			next.ServeHTTP(w, req)
			return
		}
		idToken, err := verifier.Verify(ctx, idCookie.Value)
		if err != nil {
			logger.Debug("User ID token was invalid, redircting", slogor.Err(err))
			http.Redirect(w, req, loginPath, http.StatusFound)
			return
		}
		logger.Debug("User was authenticated", slog.String("subject", idToken.Subject))
		newCtx := context.WithValue(req.Context(), authKey{}, &UserInfo{Subject: idToken.Subject})
		next.ServeHTTP(w, req.WithContext(newCtx))
	}), nil
}
