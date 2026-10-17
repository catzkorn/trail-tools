-- Links that let someone record a lactate test for the user, like a friend
-- holding the lactate meter, without signing in. Only hashes of the secrets
-- are stored: the link's token, and the secret of the cookie of the browser
-- that opened the link, which then records the test.
create table lactate_test_shares (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  token_sha256 bytea unique not null,
  -- Null until a browser opens the link.
  recorder_sha256 bytea unique,
  -- The recorded test, null until the recorder saves the first reading.
  test_id uuid references lactate_tests(id) on delete set null,
  expire_time timestamptz not null,
  create_time timestamptz not null default current_timestamp
);

create index lactate_test_shares_user on lactate_test_shares (user_id);
