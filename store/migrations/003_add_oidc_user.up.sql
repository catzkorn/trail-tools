create table oidc_users (
  id uuid primary key references users(id)
    on delete cascade
    on update cascade,
  subject text unique not null
);

create trigger insert_users_subtype before insert on oidc_users
  for each row execute procedure insert_users_subtype();
