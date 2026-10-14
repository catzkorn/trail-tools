package store

import (
	"strings"
	"testing"
)

func TestQuerySpanName(t *testing.T) {
	t.Parallel()
	long := "select " + strings.Repeat("a, ", 30) + "b from t"
	for stmt, want := range map[string]string{
		"-- name: GetRoute :one\nselect * from routes where id = $1": "GetRoute",
		"  -- name: ListRoutes :many\nselect 1":                      "ListRoutes",
		"select count(*) from routes\nwhere user_id = $1":            "select count(*) from routes",
		long: long[:maxSpanNameLength] + "…",
	} {
		if got := querySpanName(stmt); got != want {
			t.Errorf("querySpanName(%q) = %q, want %q", stmt, got, want)
		}
	}
}
