package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users/internal/gen"
	"github.com/jackc/pgx/v5/pgtype"
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
	userID, err := querier.GetSessionUser(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	retUser, err := r.loadUser(ctx, querier, userID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return retUser, nil
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
