-- name: CreateWebAuthnUser :one
insert into web_authn_users (name) values ($1) returning *;

-- name: GetWebAuthnUser :one
select * from web_authn_users where web_authn_user_id = $1;

-- name: CreateWebAuthnSession :one
insert into web_authn_sessions (
  challenge,
  relying_party_id,
  web_authn_user_id,
  allowed_credential_ids,
  expires,
  user_verification,
  extensions
) values (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7
)
-- Replace the existing session if it exists.
on conflict (web_authn_user_id) do update set
  challenge = excluded.challenge,
  relying_party_id = excluded.relying_party_id,
  allowed_credential_ids = excluded.allowed_credential_ids,
  expires = excluded.expires,
  user_verification = excluded.user_verification,
  extensions = excluded.extensions
returning *;

-- name: GetWebAuthnSession :one
delete from web_authn_sessions where web_authn_user_id = $1 returning *;

-- name: UpsertWebAuthnCredential :one
insert into web_authn_credentials (
  web_authn_user_id,
  id,
  public_key,
  attestation_type,
  transport,
  flag_user_present,
  flag_user_verified,
  flag_backup_eligible,
  flag_backup_state,
  authenticator_aaguid,
  authenticator_sign_count,
  authenticator_clone_warning,
  authenticator_attachment,
  attestation_client_data_json,
  attestation_client_data_hash,
  attestation_authenticator_data,
  attestation_public_key_algorithm,
  attestation_object
) values (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10,
  $11,
  $12,
  $13,
  $14,
  $15,
  $16,
  $17,
  $18
) on conflict (id) do update set
  web_authn_user_id = excluded.web_authn_user_id,
  public_key = excluded.public_key,
  attestation_type = excluded.attestation_type,
  transport = excluded.transport,
  flag_user_present = excluded.flag_user_present,
  flag_user_verified = excluded.flag_user_verified,
  flag_backup_eligible = excluded.flag_backup_eligible,
  flag_backup_state = excluded.flag_backup_state,
  authenticator_aaguid = excluded.authenticator_aaguid,
  authenticator_sign_count = excluded.authenticator_sign_count,
  authenticator_clone_warning = excluded.authenticator_clone_warning,
  authenticator_attachment = excluded.authenticator_attachment,
  attestation_client_data_json = excluded.attestation_client_data_json,
  attestation_client_data_hash = excluded.attestation_client_data_hash,
  attestation_authenticator_data = excluded.attestation_authenticator_data,
  attestation_public_key_algorithm = excluded.attestation_public_key_algorithm,
  attestation_object = excluded.attestation_object
returning *;

-- name: ListWebAuthnCredentials :many
select * from web_authn_credentials where web_authn_user_id = $1;
