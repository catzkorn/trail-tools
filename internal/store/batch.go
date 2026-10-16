package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// Querier runs statements. sqlc's queries run on one, e.g. gen.New(db).
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Batch queues the statements of sqlc's queries, to send them to the
// database together, in a single round trip. Use it in place of the
// connection sqlc's queries run on, e.g. gen.New(&batch), to queue
// statements whose results aren't needed, and Read to queue queries whose
// results are. Unless the batch is sent in a transaction, the statements run
// in an implicit transaction: if one fails, none take effect.
type Batch struct {
	pgx.Batch
}

// Exec queues a statement.
func (b *Batch) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	b.Queue(sql, args...)
	return pgconn.CommandTag{}, nil
}

// Query fails, as the rows wouldn't be read. Use Read instead.
func (b *Batch) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errBatchQuery
}

// QueryRow queues a statement. The returned row scans nothing.
func (b *Batch) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	b.Queue(sql, args...)
	return unreadRow{}
}

// Read queues the statement that query runs, to read its results, e.g.
//
//	b.Read(func(db store.Querier) error {
//		var err error
//		plan, err = gen.New(db).GetPlan(ctx, id)
//		return err
//	})
//
// query is called twice: first to queue its statement, on a Querier whose
// statements return no rows, and again once the batch is sent, on a Querier
// whose statement returns the queued statement's results. So query must run
// exactly one statement, and only store its results.
//
// If query returns an error once the batch is sent, the results of the
// statements after it aren't read, and sending the batch returns the error.
func (b *Batch) Read(query func(db Querier) error) {
	var r recorder
	_ = query(&r)
	if r.queued == nil {
		panic("store: Batch.Read: query ran no statement")
	}
	qq := b.Queue(r.queued.sql, r.queued.args...)
	switch r.queued.kind {
	case execKind:
		qq.Exec(func(ct pgconn.CommandTag) error { return query(replay{ct: ct}) })
	case queryKind:
		qq.Query(func(rows pgx.Rows) error { return query(replay{rows: rows}) })
	case queryRowKind:
		qq.QueryRow(func(row pgx.Row) error { return query(replay{row: row}) })
	}
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

type statementKind int

const (
	execKind statementKind = iota
	queryKind
	queryRowKind
)

// recorder is the Querier of a query being queued. It records the query's
// statement and returns no rows.
type recorder struct {
	queued *statement
}

type statement struct {
	kind statementKind
	sql  string
	args []any
}

func (r *recorder) record(kind statementKind, sql string, args []any) {
	if r.queued != nil {
		panic("store: Batch.Read: query ran more than one statement")
	}
	r.queued = &statement{kind: kind, sql: sql, args: args}
}

func (r *recorder) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	r.record(execKind, sql, args)
	return pgconn.CommandTag{}, nil
}

func (r *recorder) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	r.record(queryKind, sql, args)
	return noRows{}, nil
}

func (r *recorder) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	r.record(queryRowKind, sql, args)
	return unreadRow{}
}

// replay is the Querier of a sent query. It returns the results of the
// query's statement.
type replay struct {
	ct   pgconn.CommandTag
	rows pgx.Rows
	row  pgx.Row
}

func (r replay) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return r.ct, nil
}

func (r replay) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return r.rows, nil
}

func (r replay) QueryRow(context.Context, string, ...any) pgx.Row {
	return r.row
}

// noRows are the rows of a query being queued.
type noRows struct{}

func (noRows) Close()                                       {}
func (noRows) Err() error                                   { return nil }
func (noRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (noRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (noRows) Next() bool                                   { return false }
func (noRows) Scan(...any) error                            { return nil }
func (noRows) Values() ([]any, error)                       { return nil, nil }
func (noRows) RawValues() [][]byte                          { return nil }
func (noRows) Conn() *pgx.Conn                              { return nil }
func (noRows) TypeMap() *pgtype.Map                         { return nil }
