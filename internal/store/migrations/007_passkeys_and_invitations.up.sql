-- Passkeys are now verified with github.com/go-passkeys/go-passkeys, and
-- users can only register with an invitation.

alter table users add column is_admin boolean not null default false;

create table passkey_users (
  id uuid primary key references users(id)
    on delete cascade
    on update cascade,
  -- The WebAuthn user handle, identifying the user to authenticators.
  user_handle bytea unique not null,
  name text not null
);

-- Keep existing WebAuthn users and their data. Their credentials can't be
-- converted to the new format, so they need an invitation to add a passkey.
insert into passkey_users (id, user_handle, name)
select id, web_authn_user_id, name from web_authn_users;

-- Create the trigger after migrating, so existing users keep their IDs.
create trigger insert_users_subtype before insert on passkey_users
  for each row execute procedure insert_users_subtype();

drop table web_authn_credentials;
drop table web_authn_users;
drop type web_authn_authenticator_transport;
drop type web_authn_authenticator_attachment;

create table passkeys (
  -- The credential ID.
  id bytea primary key,
  user_id uuid not null references users(id) on delete cascade,
  -- The authenticator name, e.g. "iCloud Keychain".
  name text not null,
  -- The PKIX encoded public key.
  public_key bytea not null,
  -- The COSE algorithm identifier.
  algorithm integer not null,
  aaguid bytea not null,
  transports text[] not null,
  backup_eligible boolean not null,
  backed_up boolean not null,
  sign_count bigint not null,
  create_time timestamptz not null default current_timestamp,
  last_used_time timestamptz
);

create index passkeys_user on passkeys (user_id);

create type invitation_kind as enum (
  'new_user',
  'add_passkey'
);

create table invitations (
  id uuid primary key default gen_random_uuid(),
  token_sha256 bytea unique not null,
  kind invitation_kind not null,
  created_by uuid not null references users(id) on delete cascade,
  -- The user to add a passkey to, for add_passkey invitations.
  user_id uuid references users(id) on delete cascade,
  note text not null,
  create_time timestamptz not null default current_timestamp,
  expire_time timestamptz not null,
  use_time timestamptz
);

create type passkey_ceremony as enum (
  'registration',
  'login'
);

create table passkey_challenges (
  id uuid primary key default gen_random_uuid(),
  ceremony passkey_ceremony not null,
  challenge bytea not null,
  -- For registrations: the user handle of the new credential, and either the
  -- invitation, the signed in user adding a passkey, or the admin bootstrap.
  user_handle bytea,
  invitation_id uuid references invitations(id) on delete cascade,
  user_id uuid references users(id) on delete cascade,
  bootstrap boolean not null default false,
  expire_time timestamptz not null
);
