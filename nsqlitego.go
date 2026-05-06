// Package nsqlitego provides a NSQLite driver compatible with the database/sql
// package.
package nsqlitego

import (
	"database/sql"
	"database/sql/driver"

	"github.com/varavelio/nsqlitego/internal/nsqlitedriver"
	"github.com/varavelio/nsqlitego/nsqlitehttp"
)

// init registers the NSQLite driver for database/sql.
func init() {
	sql.Register("nsqlite", &Driver{})
}

// Driver implements database/sql/driver.Driver for NSQLite.
type Driver = nsqlitedriver.Driver

// NewConnector returns a new NSQLite connector compatible with
// database/sql.OpenDB
//
// This is the recommended way to create *sql.DB instances if you
// want to tune the underlying nsqlitehttp.Client.
func NewConnector(nsqliteHTTPClient *nsqlitehttp.Client) driver.Connector {
	return nsqlitedriver.NewConnector(nsqliteHTTPClient)
}
