package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johanbrandhorst/trail-tools/internal/store"
	"github.com/johanbrandhorst/trail-tools/internal/users/internal/gen"
)

func (r *Repository) GetSession(ctx context.Context, sessionID string) (User, error) {
	id, err := store.StringToUUID(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %w", err)
	}
	// Every signed in request gets its session, so it's read with its user in
	// a single query, without a transaction, which would take two more round
	// trips.
	row, err := r.querier.GetSessionUser(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &PasskeyUser{id: row.ID, Name: row.Name, UserHandle: row.UserHandle, AvatarKey: row.AvatarKey.String, admin: row.IsAdmin}, nil
}

func (r *Repository) DeleteSession(ctx context.Context, sessionID string) error {
	id, err := store.StringToUUID(sessionID)
	if err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}
	if err := r.querier.DeleteSession(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return store.ErrNotFound
		}
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// loadUser loads a user.
func (r *Repository) loadUser(ctx context.Context, querier *gen.Queries, userID pgtype.UUID) (User, error) {
	row, err := querier.GetPasskeyUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &PasskeyUser{id: row.ID, Name: row.Name, UserHandle: row.UserHandle, AvatarKey: row.AvatarKey.String, admin: row.IsAdmin}, nil
}
