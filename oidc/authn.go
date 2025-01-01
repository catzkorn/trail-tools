package oidc

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"gitlab.com/greyxor/slogor"
)

// UserInfo is constructed from the default OIDC "profile" claims
// Source: https://openid.net/specs/openid-connect-core-1_0.html#ScopeClaims
type UserInfo struct {
	// Standard claims
	Issuer          string
	Audience        []string
	Subject         string
	Email           string
	EmailVerified   bool
	IssuedAt        time.Time
	Expiry          time.Time
	AccessTokenHash string

	// Optional claims
	Nonce             string
	Azp               string `json:"azp"`
	Name              string `json:"name"`
	AvatarURL         string `json:"picture"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
	MiddleName        string `json:"middle_name"`
	Nickname          string `json:"nickname"`
	PreferredUsername string `json:"preferred_username"`
	Profile           string `json:"profile"`
	Website           string `json:"website"`
	Gender            string `json:"gender"`
	Zoneinfo          string `json:"zoneinfo"`
	Locale            string `json:"locale"`
	Birthdate         string `json:"birthdate"`
}

type authKey struct{}

func GetUserInfo(ctx context.Context) (UserInfo, bool) {
	userInfo, ok := ctx.Value(authKey{}).(UserInfo)
	return userInfo, ok
}

func NewAuthnMiddleware(ctx context.Context, log *slog.Logger, issuerURL string, clientID string, next http.Handler) (http.Handler, error) {
	switch {
	case log == nil:
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
			log.DebugContext(req.Context(), "User had no token, continue unauthenticated", slogor.Err(err))
			next.ServeHTTP(w, req)
			return
		}
		idToken, err := verifier.Verify(ctx, idCookie.Value)
		if err != nil {
			log.DebugContext(req.Context(), "User ID token was invalid, redirecting", slogor.Err(err))
			http.Redirect(w, req, loginPath, http.StatusFound)
			return
		}
		log.DebugContext(req.Context(), "User was authenticated", slog.String("subject", idToken.Subject))
		userInfo := UserInfo{
			Expiry:          idToken.Expiry,
			IssuedAt:        idToken.IssuedAt,
			Subject:         idToken.Subject,
			Issuer:          idToken.Issuer,
			Audience:        idToken.Audience,
			AccessTokenHash: idToken.AccessTokenHash,
			Nonce:           idToken.Nonce,
		}
		if err := idToken.Claims(&userInfo); err != nil {
			log.ErrorContext(req.Context(), "Failed to parse user info from ID token", slogor.Err(err))
			http.Redirect(w, req, loginPath, http.StatusFound)
			return
		}
		newCtx := context.WithValue(req.Context(), authKey{}, userInfo)
		next.ServeHTTP(w, req.WithContext(newCtx))
	}), nil
}
