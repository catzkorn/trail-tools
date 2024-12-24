// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package athletes

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

type Activity struct {
	ID         pgtype.UUID
	AthleteID  pgtype.UUID
	CreateTime pgtype.Timestamptz
	Name       string
}

type Athlete struct {
	ID         pgtype.UUID
	UserID     pgtype.UUID
	CreateTime pgtype.Timestamptz
	Name       string
}

type BloodLactateMeasure struct {
	ID           pgtype.UUID
	ActivityID   pgtype.UUID
	CreateTime   pgtype.Timestamptz
	MmolPerLiter decimal.Decimal
	HeartRateBpm int32
}

type User struct {
	ID          pgtype.UUID
	CreateTime  pgtype.Timestamptz
	OidcSubject string
}
