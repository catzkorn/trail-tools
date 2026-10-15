package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Batch queues the statements of sqlc's queries, to send them to the
// database together, in a single round trip. Use it in place of the
// connection sqlc's queries run on, e.g. gen.New(&batch). The statements run
// in an implicit transaction: if one fails, none take effect.
//
// The results of the statements aren't read: the queries return zero values
// and no rows, so only queue queries whose results aren't needed.
type Batch struct {
	pgx.Batch
}

// Exec queues a statement.
func (b *Batch) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	b.Queue(sql, args...)
	return pgconn.CommandTag{}, nil
}

// Query fails, as the rows wouldn't be read.
func (b *Batch) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errBatchQuery
}

// QueryRow queues a statement. The returned row scans nothing.
func (b *Batch) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	b.Queue(sql, args...)
	return unreadRow{}
}

// Send sends the queued statements to the database and waits for them to
// run.
func (b *Batch) Send(ctx context.Context, db *DB) error {
	return db.SendBatch(ctx, &b.Batch).Close()
}

var errBatchQuery = errors.New("queries returning rows can't be batched")

// unreadRow is the row of a queued statement, whose result isn't read.
type unreadRow struct{}

func (unreadRow) Scan(...any) error { return nil }
