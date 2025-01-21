create extension if not exists pgcrypto;

create table web_authn_users (
  id uuid primary key references users(id)
    on delete cascade
    on update cascade,
  web_authn_user_id bytea unique not null default gen_random_bytes(16),
  name text not null
);

create trigger insert_users_subtype before insert on web_authn_users
  for each row execute procedure insert_users_subtype();

-- https://www.w3.org/TR/web_authn-3/#enum-transport
create type web_authn_authenticator_transport as enum (
  'usb',
  'nfc',
  'ble',
  'smart-card',
  'hybrid',
  'internal'
);

-- https://www.w3.org/TR/web_authn-3/#enumdef-authenticatorattachment
create type web_authn_authenticator_attachment as enum (
  'platform',
  'cross-platform'
);

create table web_authn_credentials (
  web_authn_user_id bytea not null references web_authn_users(web_authn_user_id),
  create_time timestamptz not null default current_timestamp,
  id bytea primary key,
  public_key bytea not null,
  attestation_type text not null,
  transport web_authn_authenticator_transport array not null,
  flag_user_present bool not null,
  flag_user_verified bool not null,
  flag_backup_eligible bool not null,
  flag_backup_state bool not null,
  authenticator_aaguid bytea not null,
  -- bigint so we can fit entire uint32 range
  authenticator_sign_count bigint not null,
  authenticator_clone_warning bool not null,
  authenticator_attachment web_authn_authenticator_attachment not null,
  attestation_client_data_json bytea not null,
  attestation_client_data_hash bytea not null,
  attestation_authenticator_data bytea not null,
  attestation_public_key_algorithm bigint not null,
  attestation_object bytea not null
);
