package users

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgtype"
)

// CreateWebAuthnUser creates a new WebAuthn user with the provided name.
func (r *Repository) CreateWebAuthnUser(ctx context.Context, name string) (webauthn.User, error) {
	wUser, err := r.querier.CreateWebAuthnUser(ctx, name)
	if err != nil {
		fmt.Errorf("failed to create WebAuthn user in DB: %w", err)
	}
	user, err := newWebAuthnUser(ctx, wUser.Name, wUser.WebAuthnUserID, r.querier)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn user: %w", err)
	}
	return user, nil
}

// GetWebAuthnUser gets a WebAuthn user by ID. If the user does not exist, it returns store.ErrNotFound.
func (r *Repository) GetWebAuthnUser(ctx context.Context, webAuthnUserID []byte) (webauthn.User, error) {
	wUser, err := r.querier.GetWebAuthnUser(ctx, webAuthnUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get WebAuthn user: %w", err)
	}
	user, err := newWebAuthnUser(ctx, wUser.Name, wUser.WebAuthnUserID, r.querier)
	if err != nil {
		return nil, fmt.Errorf("failed to create WebAuthn user: %w", err)
	}
	return user, nil
}

func (r *Repository) CreateWebAuthnSession(ctx context.Context, session *webauthn.SessionData) error {
	ur := WebAuthnUserVerificationRequirement(session.UserVerification)
	if !ur.Valid() {
		return fmt.Errorf("invalid user verification requirement: %s", ur)
	}
	var extensions []byte
	if len(session.Extensions) != 0 {
		var err error
		extensions, err = json.Marshal(session.Extensions)
		if err != nil {
			return fmt.Errorf("failed to marshal extensions to JSON: %w", err)
		}
	}
	_, err := r.querier.CreateWebAuthnSession(ctx, &CreateWebAuthnSessionParams{
		Challenge:            session.Challenge,
		RelyingPartyID:       session.RelyingPartyID,
		WebAuthnUserID:       session.UserID,
		AllowedCredentialIds: session.AllowedCredentialIDs,
		Expires:              pgtype.Timestamptz{Time: session.Expires, Valid: true},
		UserVerification:     ur,
		Extensions:           extensions,
	})
	if err != nil {
		return fmt.Errorf("failed to create WebAuthn session: %w", err)
	}
	return nil
}

func (r *Repository) GetWebAuthnSession(ctx context.Context, webAuthnUserID []byte) (*webauthn.SessionData, error) {
	webAuthnSession, err := r.querier.GetWebAuthnSession(ctx, webAuthnUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get WebAuthn session: %w", err)
	}
	var extensions map[string]any
	if len(webAuthnSession.Extensions) != 0 {
		if err := json.Unmarshal(webAuthnSession.Extensions, &extensions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal extensions: %w", err)
		}
	}
	return &webauthn.SessionData{
		Challenge:            webAuthnSession.Challenge,
		RelyingPartyID:       webAuthnSession.RelyingPartyID,
		UserID:               webAuthnSession.WebAuthnUserID,
		AllowedCredentialIDs: webAuthnSession.AllowedCredentialIds,
		Expires:              webAuthnSession.Expires.Time,
		UserVerification:     protocol.UserVerificationRequirement(webAuthnSession.UserVerification),
		Extensions:           extensions,
	}, nil
}

func (r *Repository) UpsertWebAuthnCredential(ctx context.Context, webAuthnUserID []byte, credential *webauthn.Credential) error {
	var transport []WebAuthnAuthenticatorTransport
	for _, tp := range credential.Transport {
		t := WebAuthnAuthenticatorTransport(tp)
		if !t.Valid() {
			return fmt.Errorf("invalid authenticator transport: %s", t)
		}
		transport = append(transport, t)
	}
	a := WebAuthnAuthenticatorAttachment(credential.Authenticator.Attachment)
	if !a.Valid() {
		return fmt.Errorf("invalid authenticator attachment: %s", a)
	}
	_, err := r.querier.UpsertWebAuthnCredential(ctx, &UpsertWebAuthnCredentialParams{
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
