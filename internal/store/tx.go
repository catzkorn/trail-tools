package store

import (
	"context"
	"fmt"
)

// Transact sends batches one after the other, in a transaction. Each batch
// is sent in a single round trip, together with the start or end of the
// transaction if it's the first or last. Each function queues the statements
// of a batch. It's called once the results of the previous batch are read, so
// its statements can depend on them. If it returns an error, the transaction
// is rolled back and the error returned.
func (db *DB) Transact(ctx context.Context, batches ...func(b *Batch) error) error {
	return db.sendInTx(ctx, "begin", batches)
}

// ReadSnapshot is like Transact, but the transaction is read only, and reads
// from a single snapshot of the database.
func (db *DB) ReadSnapshot(ctx context.Context, batches ...func(b *Batch) error) error {
	return db.sendInTx(ctx, "begin isolation level repeatable read read only", batches)
}

func (db *DB) sendInTx(ctx context.Context, begin string, batches []func(b *Batch) error) error {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()
	defer func() {
		// The transaction is left open if a batch failed or wasn't sent. The
		// pool would close the connection rather than reuse it, so roll the
		// transaction back.
		if pg := conn.Conn().PgConn(); !pg.IsClosed() && pg.TxStatus() != 'I' {
			_, _ = conn.Exec(context.Background(), "rollback")
		}
	}()
	for i, queue := range batches {
		var b Batch
		if i == 0 {
			b.Queue(begin)
		}
		if err := queue(&b); err != nil {
			return err
		}
		if i == len(batches)-1 {
			b.Queue("commit")
		}
		if err := conn.SendBatch(ctx, &b.Batch).Close(); err != nil {
			return err
		}
	}
	return nil
}
