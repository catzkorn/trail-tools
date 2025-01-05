// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: users.sql

package users

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const getUser = `-- name: GetUser :one
select id, create_time from users where id = $1
`

func (q *Queries) GetUser(ctx context.Context, id pgtype.UUID) (*User, error) {
	row := q.db.QueryRow(ctx, getUser, id)
	var i User
	err := row.Scan(&i.ID, &i.CreateTime)
	return &i, err
}
