package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/catzkorn/trail-tools/internal/users/internal"
)

type OIDCUser struct {
	*internal.OIDCUser
}

// GetOIDCUser gets or creates a new OIDC user with the provided subject.
func (r *Repository) GetOIDCUser(ctx context.Context, subject string) (*OIDCUser, error) {
	oidcUser, err := r.querier.GetOIDCUser(ctx, subject)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		oidcUser, err = r.querier.CreateOIDCUser(ctx, subject)
		if err != nil {
			fmt.Errorf("failed to create new oidc user: %w", err)
		}
	case err != nil:
		return nil, fmt.Errorf("failed to get oidc user: %w", err)
	default:
		// User retrieved successfully
	}
	return &OIDCUser{
		OIDCUser: oidcUser,
	}, nil
}
