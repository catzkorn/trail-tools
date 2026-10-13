-- The key of the user's avatar image in the blob store, if they uploaded
-- one. Each upload gets a new key, so avatar images can be cached forever.
alter table users
  add column avatar_key text;
