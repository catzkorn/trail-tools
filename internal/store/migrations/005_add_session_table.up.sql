create table sessions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id)
    on delete cascade
    on update cascade,
  expiry timestamptz not null
    check (expiry > now())
);
