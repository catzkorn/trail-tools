package authn

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"gitlab.com/greyxor/slogor"
)

// googleIssuer is the issuer of Google signed ID tokens.
const googleIssuer = "https://accounts.google.com"

// RequireGoogleServiceAccount returns a handler that only allows requests
// with a Google signed ID token for the service account and audience, such
// as requests from Cloud Scheduler.
func RequireGoogleServiceAccount(ctx context.Context, log *slog.Logger, audience, email string, next http.Handler) (http.Handler, error) {
	provider, err := oidc.NewProvider(ctx, googleIssuer)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google OIDC provider: %w", err)
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: audience})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			http.Error(w, "missing ID token", http.StatusUnauthorized)
			return
		}
		idToken, err := verifier.Verify(r.Context(), token)
		if err != nil {
			log.WarnContext(r.Context(), "Invalid Google ID token", slogor.Err(err))
			http.Error(w, "invalid ID token", http.StatusUnauthorized)
			return
		}
		var claims struct {
			Email         string `json:"email"`
			EmailVerified bool   `json:"email_verified"`
		}
		if err := idToken.Claims(&claims); err != nil || !claims.EmailVerified || claims.Email != email {
			log.WarnContext(r.Context(), "ID token for unexpected account", slog.String("email", claims.Email))
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}), nil
}
