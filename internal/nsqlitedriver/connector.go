package nsqlitedriver

import (
	"context"
	"database/sql/driver"

	"github.com/varavelio/nsqlitego/nsqlitehttp"
)

var _ driver.Connector = (*nsqliteConnector)(nil)

// nsqliteConnector represents a driver in a fixed configuration and can create
// any number of equivalent Conns for use by multiple goroutines.
type nsqliteConnector struct {
	httpClient *nsqlitehttp.Client
}

// NewConnector returns a new NSQLite connector compatible with
// database/sql.OpenDB
//
// It accepts a number of options to configure the connector.
func NewConnector(nsqliteHTTPClient *nsqlitehttp.Client) driver.Connector {
	connector := &nsqliteConnector{
		httpClient: nsqliteHTTPClient,
	}

	return connector
}

// Connect returns a connection to the database.
func (c *nsqliteConnector) Connect(_ context.Context) (driver.Conn, error) {
	return &Conn{client: c.httpClient}, nil
}

// Driver returns the underlying Driver of the Connector
func (c *nsqliteConnector) Driver() driver.Driver {
	return &Driver{}
}
