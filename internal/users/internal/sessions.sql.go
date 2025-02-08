// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: sessions.sql

package internal

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createSession = `-- name: CreateSession :one
insert into sessions (user_id, expiry) values ($1, $2) returning id
`

type CreateSessionParams struct {
	UserID pgtype.UUID
	Expiry pgtype.Timestamptz
}

func (q *Queries) CreateSession(ctx context.Context, arg *CreateSessionParams) (pgtype.UUID, error) {
	row := q.db.QueryRow(ctx, createSession, arg.UserID, arg.Expiry)
	var id pgtype.UUID
	err := row.Scan(&id)
	return id, err
}

const getSessionUser = `-- name: GetSessionUser :one
select
  user_id,
  case 
    when exists(select 1 from oidc_users where sessions.user_id=oidc_users.id) then 'oidc'
    when exists(select 1 from web_authn_users where sessions.user_id=web_authn_users.id) then 'webauthn'
  end::text as type
from sessions where sessions.id = $1 and sessions.expiry > now()
`

type GetSessionUserRow struct {
	UserID pgtype.UUID
	Type   string
}

func (q *Queries) GetSessionUser(ctx context.Context, id pgtype.UUID) (*GetSessionUserRow, error) {
	row := q.db.QueryRow(ctx, getSessionUser, id)
	var i GetSessionUserRow
	err := row.Scan(&i.UserID, &i.Type)
	return &i, err
}
