package nsqlitedriver

import (
	"context"
	"database/sql/driver"
	"fmt"
	"sync"

	"github.com/varavelio/nsqlitego/nsqlitehttp"
)

var (
	_ driver.Driver        = (*Driver)(nil)
	_ driver.DriverContext = (*Driver)(nil)
)

// Driver implements database/sql/driver.Driver for NSQLite.
type Driver struct{}

var (
	httpClient     *nsqlitehttp.Client
	httpClientErr  error
	httpClientOnce sync.Once
)

// getNSQLiteHTTPClient creates a new NSQLite HTTP client singleton.
func getNSQLiteHTTPClient(connectionString string) (*nsqlitehttp.Client, error) {
	httpClientOnce.Do(func() {
		hc, err := nsqlitehttp.NewClient(connectionString)
		if err != nil {
			httpClientErr = fmt.Errorf("failed to create NSQLite HTTP client: %w", err)
			return
		}

		httpClient = hc
		httpClientErr = nil
	})

	return httpClient, httpClientErr
}

// Open creates a new connection using the provided connection string.
func (d *Driver) Open(connectionString string) (driver.Conn, error) {
	httpClient, err := getNSQLiteHTTPClient(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to create NSQLite HTTP client: %w", err)
	}

	connector := NewConnector(httpClient)
	return connector.Connect(context.Background())
}

// OpenConnector creates a new connector using the provided connection string.
func (d *Driver) OpenConnector(connectionString string) (driver.Connector, error) {
	httpClient, err := getNSQLiteHTTPClient(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to create NSQLite HTTP client: %w", err)
	}

	return NewConnector(httpClient), nil
}
