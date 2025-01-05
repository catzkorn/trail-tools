package users

import (
	"context"
	"fmt"

	"github.com/catzkorn/trail-tools/internal/users/internal"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type webAuthnUser struct {
	name  string
	id    []byte
	creds []webauthn.Credential
}

func newWebAuthnUser(ctx context.Context, name string, id []byte, querier *internal.Queries) (*webAuthnUser, error) {
	dbCreds, err := querier.ListWebAuthnCredentials(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to list webauthn credentials for user %q: %w", string(id), err)
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
	return &webAuthnUser{
		name:  name,
		id:    id,
		creds: creds,
	}, nil
}

func (w *webAuthnUser) WebAuthnID() []byte {
	return w.id
}

func (w *webAuthnUser) WebAuthnName() string {
	return w.name
}

func (w *webAuthnUser) WebAuthnDisplayName() string {
	return w.name
}

func (w *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return w.creds
}
