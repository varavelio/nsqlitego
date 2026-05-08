package nsqlitedriver

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/varavelio/nsqlitego/nsqlitehttp"
)

var (
	_ driver.Conn               = (*Conn)(nil)
	_ driver.ConnBeginTx        = (*Conn)(nil)
	_ driver.ConnPrepareContext = (*Conn)(nil)
	_ driver.Pinger             = (*Conn)(nil)
	_ driver.SessionResetter    = (*Conn)(nil)
	_ driver.Validator          = (*Conn)(nil)
)

// Conn represents a connection to the NSQLite server.
type Conn struct {
	// client is the HTTP client used to communicate with the NSQLite server.
	client *nsqlitehttp.Client

	// txID is the ID of the current transaction, if empty no transaction is
	// active.
	//
	// We can store this here because docs say that the connection is not used
	// concurrently by multiple goroutines and is assumed to be stateful.
	//
	// https://pkg.go.dev/database/sql/driver#Conn
	//
	// Stmt is where we use txID, and because Stmt is not used concurrently and
	// is bound to a Conn, we theoretically can store it here.
	//
	// https://pkg.go.dev/database/sql/driver#Stmt
	txID string
}

// Prepare creates a prepared statement with the given query.
func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

// PrepareContext creates a prepared statement with the given query and context.
func (c *Conn) PrepareContext(_ context.Context, query string) (driver.Stmt, error) {
	return &Stmt{
		conn:  c,
		query: query,
	}, nil
}

// Close closes the connection, releasing any open resources.
func (c *Conn) Close() error {
	if err := c.RollbackTx(context.Background()); err != nil {
		return fmt.Errorf("failed closing connection: %w", err)
	}
	return nil
}

// Begin starts a new transaction.
func (c *Conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

// BeginTx starts a new transaction with the provided context.
func (c *Conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	resp, err := c.client.SendQuery(ctx, nsqlitehttp.Query{
		Query: "BEGIN;",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	if resp.GetError() != "" {
		return nil, fmt.Errorf("failed to begin transaction: %s", resp.GetError())
	}
	if resp.GetTxId() == "" {
		return nil, fmt.Errorf("transaction ID not returned from server")
	}

	c.setTxId(resp.GetTxId())
	return &Tx{
		conn: c,
	}, nil
}

// CommitTx commits the transaction if any, otherwise does nothing.
func (c *Conn) CommitTx(ctx context.Context) error {
	defer c.setTxId("")
	if c.txID == "" {
		return nil
	}

	resp, err := c.client.SendQuery(ctx, nsqlitehttp.Query{
		Query: "COMMIT",
		TxId:  optionalString(c.txID),
	})
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	if resp.GetError() != "" {
		return fmt.Errorf("failed to commit transaction: %s", resp.GetError())
	}

	return nil
}

// RollbackTx rolls back the transaction if any, otherwise does nothing.
func (c *Conn) RollbackTx(ctx context.Context) error {
	defer c.setTxId("")
	if c.txID == "" {
		return nil
	}

	resp, err := c.client.SendQuery(ctx, nsqlitehttp.Query{
		Query: "ROLLBACK",
		TxId:  optionalString(c.txID),
	})
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	if resp.GetError() != "" {
		return fmt.Errorf("failed to rollback transaction: %s", resp.GetError())
	}

	return nil
}

// setTxId sets the transaction ID for the connection.
func (c *Conn) setTxId(txId string) {
	c.txID = txId
}

// Ping verifies that the connection is still alive.
func (c *Conn) Ping(ctx context.Context) error {
	return c.client.SendPing(ctx)
}

// ResetSession resets the session state used when the connection was used
// before and needs to be reused.
func (c *Conn) ResetSession(ctx context.Context) error {
	if err := c.RollbackTx(ctx); err != nil {
		return errors.Join(
			driver.ErrBadConn, errors.New("error resetting session"), err,
		)
	}
	return nil
}

// IsValid is called prior to placing the connection into the connection pool.
// The connection will be discarded if false is returned.
//
// Always returns true because the actual query will fail cleanly if the server
// is unreachable, avoiding the overhead of a health check on every connection
// reuse.
func (c *Conn) IsValid() bool {
	return true
}
