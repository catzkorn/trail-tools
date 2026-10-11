package store

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// DateLayout is the layout of dates in the API.
const DateLayout = "2006-01-02"

// ParseDate parses a YYYY-MM-DD date into a time at midnight UTC.
func ParseDate(s string) (time.Time, error) {
	t, err := time.Parse(DateLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD", s)
	}
	return t, nil
}

// FormatDate formats a date as YYYY-MM-DD. It returns an empty string for
// the zero time.
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(DateLayout)
}

// DateToPG converts a date to a postgres date. The zero time is NULL.
func DateToPG(t time.Time) pgtype.Date {
	if t.IsZero() {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: t, Valid: true}
}

// DateFromPG converts a postgres date to a time at midnight UTC. NULL is the
// zero time.
func DateFromPG(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return time.Date(d.Time.Year(), d.Time.Month(), d.Time.Day(), 0, 0, 0, 0, time.UTC)
}

// UUIDsToPG converts strings to UUIDs.
func UUIDsToPG(ids []string) ([]pgtype.UUID, error) {
	result := make([]pgtype.UUID, len(ids))
	for i, id := range ids {
		u, err := StringToUUID(id)
		if err != nil {
			return nil, fmt.Errorf("invalid ID %q: %w", id, err)
		}
		result[i] = u
	}
	return result, nil
}
