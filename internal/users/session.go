package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/catzkorn/trail-tools/internal/store"
)

func (r *Repository) GetSession(ctx context.Context, sessionID string) (User, error) {
	id, err := store.StringToUUID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %w", err)
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())
	querier := r.querier.WithTx(tx)
	user, err := querier.GetSessionUser(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	switch user.Type {
	case "oidc":
		dbUser, err := querier.GetOIDCUser(ctx, user.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get OIDC user: %w", err)
		}
		return &OIDCUser{
			OIDCUser: dbUser,
		}, nil
	case "webauthn":
		dbUser, err := querier.GetWebAuthnUser(ctx, user.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get WebAuthn user: %w", err)
		}
		webAuthnUser, err := newWebAuthnUser(ctx, dbUser.ID, dbUser.Name, dbUser.WebAuthnUserID, r.querier)
		if err != nil {
			return nil, fmt.Errorf("failed to create WebAuthn user: %w", err)
		}
		return webAuthnUser, nil
	default:
		return nil, fmt.Errorf("unknown user type %q", user.Type)
	}
}
