package store

import "github.com/jackc/pgx/v5/pgtype"

func UUIDToString(in pgtype.UUID) string {
	if !in.Valid {
		return ""
	}
	s, _ := in.Value()
	return s.(string)
}

func StringToUUID(in string) (pgtype.UUID, error) {
	u := pgtype.UUID{}
	if err := u.Scan(in); err != nil {
		return pgtype.UUID{}, err
	}
	return u, nil
}
