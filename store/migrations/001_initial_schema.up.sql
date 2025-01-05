create extension if not exists pgcrypto;

create table users (
  id uuid primary key default gen_random_uuid(),
  create_time timestamptz not null default current_timestamp
);

create or replace function insert_users_subtype() returns trigger
as $$
begin
  insert into users default values returning id into new.id;
  return new;
end;
$$ language plpgsql;
