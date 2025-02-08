package users

import (
	"context"
	"fmt"

	"github.com/catzkorn/trail-tools/internal/users/internal"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jackc/pgx/v5/pgtype"
)

type WebAuthnUser struct {
	id             pgtype.UUID
	name           string
	webAuthnUserID []byte
	creds          []webauthn.Credential
}

func newWebAuthnUser(ctx context.Context, id pgtype.UUID, name string, webAuthnUserID []byte, querier *internal.Queries) (*WebAuthnUser, error) {
	dbCreds, err := querier.ListWebAuthnCredentials(ctx, webAuthnUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list webauthn credentials: %w", err)
	}
	var creds []webauthn.Credential
	for _, cred := range dbCreds {
		var transport []protocol.AuthenticatorTransport
		for _, t := range cred.Transport {
			transport = append(transport, protocol.AuthenticatorTransport(t))
		}
		creds = append(creds, webauthn.Credential{
			ID:              cred.ID,
			PublicKey:       cred.PublicKey,
			AttestationType: cred.AttestationType,
			Transport:       transport,
			Flags: webauthn.CredentialFlags{
				UserPresent:    cred.FlagUserPresent,
				UserVerified:   cred.FlagUserVerified,
				BackupEligible: cred.FlagBackupEligible,
				BackupState:    cred.FlagBackupState,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:       cred.AuthenticatorAaguid,
				SignCount:    uint32(cred.AuthenticatorSignCount),
				CloneWarning: cred.AuthenticatorCloneWarning,
				Attachment:   protocol.AuthenticatorAttachment(cred.AuthenticatorAttachment),
			},
			Attestation: webauthn.CredentialAttestation{
				ClientDataJSON:     cred.AttestationClientDataJSON,
				ClientDataHash:     cred.AttestationClientDataHash,
				AuthenticatorData:  cred.AttestationAuthenticatorData,
				PublicKeyAlgorithm: cred.AttestationPublicKeyAlgorithm,
				Object:             cred.AttestationObject,
			},
		})
	}
	return &WebAuthnUser{
		id:             id,
		name:           name,
		webAuthnUserID: webAuthnUserID,
		creds:          creds,
	}, nil
}

func (w *WebAuthnUser) ID() pgtype.UUID {
	return w.id
}

func (w *WebAuthnUser) WebAuthnID() []byte {
	return w.webAuthnUserID
}

func (w *WebAuthnUser) WebAuthnName() string {
	return w.name
}

func (w *WebAuthnUser) WebAuthnDisplayName() string {
	return w.name
}

func (w *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return w.creds
}
