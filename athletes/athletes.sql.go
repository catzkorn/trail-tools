// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: athletes.sql

package athletes

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

const addActivity = `-- name: AddActivity :one
insert into activities (
  name,
  athlete_id
) values (
  $1,
  $2
)
returning id, athlete_id, create_time, name
`

type AddActivityParams struct {
	Name      string
	AthleteID pgtype.UUID
}

func (q *Queries) AddActivity(ctx context.Context, arg *AddActivityParams) (*Activity, error) {
	row := q.db.QueryRow(ctx, addActivity, arg.Name, arg.AthleteID)
	var i Activity
	err := row.Scan(
		&i.ID,
		&i.AthleteID,
		&i.CreateTime,
		&i.Name,
	)
	return &i, err
}

const addAthlete = `-- name: AddAthlete :one
insert into athletes (
  name,
  user_id
) values (
  $1,
  $2
)
returning id, user_id, create_time, name
`

type AddAthleteParams struct {
	Name   string
	UserID pgtype.UUID
}

func (q *Queries) AddAthlete(ctx context.Context, arg *AddAthleteParams) (*Athlete, error) {
	row := q.db.QueryRow(ctx, addAthlete, arg.Name, arg.UserID)
	var i Athlete
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.CreateTime,
		&i.Name,
	)
	return &i, err
}

const addMeasure = `-- name: AddMeasure :one
insert into blood_lactate_measures (
  activity_id,
  mmol_per_liter,
  heart_rate_bpm
) values (
  $1,
  $2,
  $3
)
returning id, activity_id, create_time, mmol_per_liter, heart_rate_bpm
`

type AddMeasureParams struct {
	ActivityID   pgtype.UUID
	MmolPerLiter decimal.Decimal
	HeartRateBpm int32
}

func (q *Queries) AddMeasure(ctx context.Context, arg *AddMeasureParams) (*BloodLactateMeasure, error) {
	row := q.db.QueryRow(ctx, addMeasure, arg.ActivityID, arg.MmolPerLiter, arg.HeartRateBpm)
	var i BloodLactateMeasure
	err := row.Scan(
		&i.ID,
		&i.ActivityID,
		&i.CreateTime,
		&i.MmolPerLiter,
		&i.HeartRateBpm,
	)
	return &i, err
}

const deleteAthlete = `-- name: DeleteAthlete :one
delete from athletes
where id = $1
returning id, user_id, create_time, name
`

func (q *Queries) DeleteAthlete(ctx context.Context, id pgtype.UUID) (*Athlete, error) {
	row := q.db.QueryRow(ctx, deleteAthlete, id)
	var i Athlete
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.CreateTime,
		&i.Name,
	)
	return &i, err
}
