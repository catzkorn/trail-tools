package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users/internal"
	"github.com/jackc/pgx/v5/pgtype"
)

type OIDCUser struct {
	*internal.OIDCUser
}

// CreateOIDCSession creates a new OIDC session using provided subject
// to identify the user. A DB entry will be created if one does not already
// exist.
func (r *Repository) CreateOIDCSession(ctx context.Context, subject string, expiry time.Time) (string, error) {
	if expiry.Before(time.Now()) {
		return "", errors.New("expiry must be in the future")
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())
	querier := r.querier.WithTx(tx)
	user, err := querier.CreateOIDCUser(ctx, subject)
	if err != nil {
		return "", fmt.Errorf("failed to create new oidc user: %w", err)
	}
	sessionID, err := querier.CreateSession(ctx, &internal.CreateSessionParams{
		UserID: user.ID,
		Expiry: pgtype.Timestamptz{
			Time:  expiry,
			Valid: true,
		},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", store.ErrNotFound
		}
		return "", fmt.Errorf("failed to create sessions: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}
	return store.UUIDToString(sessionID), nil
}

func (u *OIDCUser) ID() pgtype.UUID {
	return u.OIDCUser.ID
}
