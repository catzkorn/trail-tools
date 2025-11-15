package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users/internal"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgtype"
)

// CreateWebAuthnUser creates a new WebAuthn user with the provided name.
func (r *Repository) CreateWebAuthnUser(ctx context.Context, name string) (*WebAuthnUser, error) {
	dbUser, err := r.querier.CreateWebAuthnUser(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn user in DB: %w", err)
	}
	user, err := newWebAuthnUser(ctx, dbUser.ID, dbUser.Name, dbUser.WebAuthnUserID, r.querier)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn user: %w", err)
	}
	return user, nil
}

// GetWebAuthnUser gets a WebAuthn user by ID. If the user does not exist, it returns store.ErrNotFound.
func (r *Repository) GetWebAuthnUser(ctx context.Context, webAuthnUserID []byte) (*WebAuthnUser, error) {
	wUser, err := r.querier.GetWebAuthnUserByWebAuthnUserID(ctx, webAuthnUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get WebAuthn user: %w", err)
	}
	user, err := newWebAuthnUser(ctx, wUser.ID, wUser.Name, wUser.WebAuthnUserID, r.querier)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn user: %w", err)
	}
	return user, nil
}

func (r *Repository) UpsertWebAuthnCredential(ctx context.Context, webAuthnUserID []byte, credential *webauthn.Credential) error {
	var transport []internal.WebAuthnAuthenticatorTransport
	for _, tp := range credential.Transport {
		t := internal.WebAuthnAuthenticatorTransport(tp)
		if !t.Valid() {
			return fmt.Errorf("invalid authenticator transport: %s", t)
		}
		transport = append(transport, t)
	}
	a := internal.WebAuthnAuthenticatorAttachment(credential.Authenticator.Attachment)
	if !a.Valid() {
		return fmt.Errorf("invalid authenticator attachment: %s", a)
	}
	_, err := r.querier.UpsertWebAuthnCredential(ctx, &internal.UpsertWebAuthnCredentialParams{
		WebAuthnUserID:                webAuthnUserID,
		ID:                            credential.ID,
		PublicKey:                     credential.PublicKey,
		AttestationType:               credential.AttestationType,
		Transport:                     transport,
		FlagUserPresent:               credential.Flags.UserPresent,
		FlagUserVerified:              credential.Flags.UserVerified,
		FlagBackupEligible:            credential.Flags.BackupEligible,
		FlagBackupState:               credential.Flags.BackupState,
		AuthenticatorAaguid:           credential.Authenticator.AAGUID,
		AuthenticatorSignCount:        int64(credential.Authenticator.SignCount),
		AuthenticatorCloneWarning:     credential.Authenticator.CloneWarning,
		AuthenticatorAttachment:       a,
		AttestationClientDataJSON:     credential.Attestation.ClientDataJSON,
		AttestationClientDataHash:     credential.Attestation.ClientDataHash,
		AttestationAuthenticatorData:  credential.Attestation.AuthenticatorData,
		AttestationPublicKeyAlgorithm: credential.Attestation.PublicKeyAlgorithm,
		AttestationObject:             credential.Attestation.Object,
	})
	if err != nil {
		return fmt.Errorf("failed to create WebAuthn credential: %w", err)
	}
	return nil
}

func (r *Repository) CreateWebAuthnSession(ctx context.Context, user *WebAuthnUser, expiry time.Time) (string, error) {
	if expiry.Before(time.Now()) {
		return "", errors.New("expiry must be in the future")
	}
	sessionID, err := r.querier.CreateSession(ctx, &internal.CreateSessionParams{
		UserID: user.id,
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
	return store.UUIDToString(sessionID), nil
}
