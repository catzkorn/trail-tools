package store_test

import (
	"context"
	"testing"

	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/store/storetest"
)

func TestBatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := storetest.New(t)
	if _, err := db.Exec(ctx, "create table batched (n int primary key)"); err != nil {
		t.Fatal(err)
	}
	count := func() int {
		t.Helper()
		var n int
		if err := db.QueryRow(ctx, "select count(*) from batched").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	var batch store.Batch
	if _, err := batch.Exec(ctx, "insert into batched values ($1)", 1); err != nil {
		t.Fatal(err)
	}
	var unread int
	if err := batch.QueryRow(ctx, "insert into batched values ($1) returning n", 2).Scan(&unread); err != nil || unread != 0 {
		t.Fatalf("QueryRow scanned %d, %v; want nothing", unread, err)
	}
	if _, err := batch.Query(ctx, "select n from batched"); err == nil {
		t.Error("Query succeeded, want an error")
	}
	if err := batch.Send(ctx, db); err != nil {
		t.Fatal(err)
	}
	if n := count(); n != 2 {
		t.Fatalf("got %d rows, want 2", n)
	}

	// The statements run in a transaction: if one fails, none take effect.
	batch = store.Batch{}
	if _, err := batch.Exec(ctx, "insert into batched values ($1)", 3); err != nil {
		t.Fatal(err)
	}
	if _, err := batch.Exec(ctx, "insert into batched values ($1)", 1); err != nil {
		t.Fatal(err)
	}
	if err := batch.Send(ctx, db); err == nil {
		t.Fatal("sent a batch with a duplicate key, want an error")
	}
	if n := count(); n != 2 {
		t.Errorf("got %d rows after a failed batch, want 2", n)
	}
}
