package users

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johanbrandhorst/trail-tools/internal/store"
	"github.com/johanbrandhorst/trail-tools/internal/users/internal/gen"
)

const (
	// oauthCodePrefix identifies OAuth authorization codes.
	oauthCodePrefix = "ttac_"
	// oauthRefreshTokenPrefix identifies OAuth refresh tokens, so they can be
	// recognised if leaked. OAuth access tokens are API tokens, and use
	// apiTokenPrefix.
	oauthRefreshTokenPrefix = "ttr_"
)

// OAuthAuthorization is a user's approval of an OAuth client, exchanged for
// tokens with an authorization code.
type OAuthAuthorization struct {
	UserID        pgtype.UUID
	ClientID      string
	ClientName    string
	RedirectURI   string
	CodeChallenge string
}

// OAuthTokens are the tokens issued to an OAuth client.
type OAuthTokens struct {
	AccessToken      string
	AccessExpireTime time.Time
	RefreshToken     string
}

// OAuthGrant is a user's authorization of an OAuth client.
type OAuthGrant struct {
	*gen.ListOAuthGrantsRow
}

// CreateOAuthAuthorizationCode stores the authorization and returns an
// authorization code for it, valid until the expiry. Only a hash of the code
// is stored.
func (r *Repository) CreateOAuthAuthorizationCode(ctx context.Context, authz *OAuthAuthorization, expiry time.Time) (string, error) {
	code := oauthCodePrefix + rand.Text()
	hash := sha256.Sum256([]byte(code))
	// Clean up codes that were never used.
	if err := r.querier.DeleteExpiredOAuthAuthorizationCodes(ctx); err != nil {
		return "", fmt.Errorf("failed to delete expired authorization codes: %w", err)
	}
	err := r.querier.CreateOAuthAuthorizationCode(ctx, &gen.CreateOAuthAuthorizationCodeParams{
		CodeSHA256:    hash[:],
		UserID:        authz.UserID,
		ClientID:      authz.ClientID,
		ClientName:    authz.ClientName,
		RedirectURI:   authz.RedirectURI,
		CodeChallenge: authz.CodeChallenge,
		ExpireTime:    pgtype.Timestamptz{Time: expiry, Valid: true},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create authorization code: %w", err)
	}
	return code, nil
}

// ConsumeOAuthAuthorizationCode returns the authorization of an authorization
// code and deletes the code, so it can only be used once. It returns
// store.ErrNotFound if the code doesn't exist or has expired.
func (r *Repository) ConsumeOAuthAuthorizationCode(ctx context.Context, code string) (*OAuthAuthorization, error) {
	hash := sha256.Sum256([]byte(code))
	row, err := r.querier.ConsumeOAuthAuthorizationCode(ctx, hash[:])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to consume authorization code: %w", err)
	}
	return &OAuthAuthorization{
		UserID:        row.UserID,
		ClientID:      row.ClientID,
		ClientName:    row.ClientName,
		RedirectURI:   row.RedirectURI,
		CodeChallenge: row.CodeChallenge,
	}, nil
}

// CreateOAuthGrant grants the client of the authorization access on behalf
// of the user, and returns its first tokens.
func (r *Repository) CreateOAuthGrant(ctx context.Context, authz *OAuthAuthorization, accessExpiry, refreshExpiry time.Time) (*OAuthTokens, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())
	querier := r.querier.WithTx(tx)
	// Clean up grants whose refresh token expired, as they can't be used.
	if err := querier.DeleteExpiredOAuthGrants(ctx, authz.UserID); err != nil {
		return nil, fmt.Errorf("failed to delete expired grants: %w", err)
	}
	refreshToken := oauthRefreshTokenPrefix + rand.Text()
	refreshHash := sha256.Sum256([]byte(refreshToken))
	grant, err := querier.CreateOAuthGrant(ctx, &gen.CreateOAuthGrantParams{
		UserID:             authz.UserID,
		ClientID:           authz.ClientID,
		ClientName:         authz.ClientName,
		RedirectURI:        authz.RedirectURI,
		RefreshTokenSHA256: refreshHash[:],
		RefreshExpireTime:  pgtype.Timestamptz{Time: refreshExpiry, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create grant: %w", err)
	}
	accessToken, err := createOAuthAccessToken(ctx, querier, grant, accessExpiry)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return &OAuthTokens{
		AccessToken:      accessToken,
		AccessExpireTime: accessExpiry,
		RefreshToken:     refreshToken,
	}, nil
}

// RefreshOAuthGrant issues new tokens for the grant with the refresh token.
// The refresh token is rotated, so it can only be used once. It returns
// store.ErrNotFound if the refresh token doesn't exist, has expired, or
// belongs to another client.
func (r *Repository) RefreshOAuthGrant(ctx context.Context, refreshToken, clientID string, accessExpiry, refreshExpiry time.Time) (*OAuthTokens, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())
	querier := r.querier.WithTx(tx)
	hash := sha256.Sum256([]byte(refreshToken))
	grant, err := querier.GetOAuthGrantByRefreshToken(ctx, hash[:])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get grant: %w", err)
	}
	if grant.ClientID != clientID {
		return nil, store.ErrNotFound
	}
	newRefreshToken := oauthRefreshTokenPrefix + rand.Text()
	newHash := sha256.Sum256([]byte(newRefreshToken))
	err = querier.RotateOAuthRefreshToken(ctx, &gen.RotateOAuthRefreshTokenParams{
		ID:                 grant.ID,
		RefreshTokenSHA256: newHash[:],
		RefreshExpireTime:  pgtype.Timestamptz{Time: refreshExpiry, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to rotate refresh token: %w", err)
	}
	if err := querier.DeleteExpiredOAuthAccessTokens(ctx, grant.ID); err != nil {
		return nil, fmt.Errorf("failed to delete expired access tokens: %w", err)
	}
	accessToken, err := createOAuthAccessToken(ctx, querier, grant, accessExpiry)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return &OAuthTokens{
		AccessToken:      accessToken,
		AccessExpireTime: accessExpiry,
		RefreshToken:     newRefreshToken,
	}, nil
}

// createOAuthAccessToken creates an API token belonging to the grant, which
// expires.
func createOAuthAccessToken(ctx context.Context, querier *gen.Queries, grant *gen.OAuthGrant, expiry time.Time) (string, error) {
	secret := apiTokenPrefix + rand.Text()
	hash := sha256.Sum256([]byte(secret))
	_, err := querier.CreateOAuthAccessToken(ctx, &gen.CreateOAuthAccessTokenParams{
		UserID:       grant.UserID,
		Name:         grant.ClientName,
		SecretSHA256: hash[:],
		OAuthGrantID: grant.ID,
		ExpireTime:   pgtype.Timestamptz{Time: expiry, Valid: true},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create access token: %w", err)
	}
	return secret, nil
}

// ListOAuthGrants lists the user's unexpired OAuth grants, newest first.
func (r *Repository) ListOAuthGrants(ctx context.Context, userID pgtype.UUID) ([]*OAuthGrant, error) {
	rows, err := r.querier.ListOAuthGrants(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list OAuth grants: %w", err)
	}
	grants := make([]*OAuthGrant, len(rows))
	for i, row := range rows {
		grants[i] = &OAuthGrant{ListOAuthGrantsRow: row}
	}
	return grants, nil
}

// DeleteOAuthGrant deletes one of the user's OAuth grants, revoking its
// tokens.
func (r *Repository) DeleteOAuthGrant(ctx context.Context, userID pgtype.UUID, grantID pgtype.UUID) error {
	n, err := r.querier.DeleteOAuthGrant(ctx, &gen.DeleteOAuthGrantParams{
		ID:     grantID,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete OAuth grant: %w", err)
	}
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}
