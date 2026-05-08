package nsqlitehttp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/varavelio/nsqlitego/internal/vdl"
	"github.com/varavelio/nsqlitego/nsqlitedsn"
)

type (
	Query               = vdl.Query
	QueryParam          = vdl.QueryParam
	QueryResult         = vdl.QueryResult
	QueryResponse       = vdl.QueryResult
	QueryResponseType   = vdl.QueryResultType
	SqliteValue         = vdl.SqliteValue
	SqliteStorageClass  = vdl.SqliteStorageClass
	Stats               = vdl.Stats
	StatsTotals         = vdl.StatsTotalsCounters
	StatsQueued         = vdl.StatsQueuedCounters
	SystemHealthOutput  = vdl.SystemHealthOutput
	SystemSessionOutput = vdl.SystemSessionOutput
	SystemStatusOutput  = vdl.SystemStatusOutput
	AuthRole            = vdl.AuthRole
)

const (
	QueryResponseTypeError    = vdl.QueryResultTypeError
	QueryResponseTypeBegin    = vdl.QueryResultTypeBegin
	QueryResponseTypeCommit   = vdl.QueryResultTypeCommit
	QueryResponseTypeRollback = vdl.QueryResultTypeRollback
	QueryResponseTypeWrite    = vdl.QueryResultTypeWrite
	QueryResponseTypeRead     = vdl.QueryResultTypeRead

	SqliteStorageClassNull    = vdl.SqliteStorageClassNull
	SqliteStorageClassInteger = vdl.SqliteStorageClassInteger
	SqliteStorageClassReal    = vdl.SqliteStorageClassReal
	SqliteStorageClassText    = vdl.SqliteStorageClassText
	SqliteStorageClassBlob    = vdl.SqliteStorageClassBlob

	AuthRoleAdmin     = vdl.AuthRoleAdmin
	AuthRoleReadWrite = vdl.AuthRoleReadWrite
	AuthRoleReadOnly  = vdl.AuthRoleReadOnly
)

// Client is an HTTP client for the NSQLite server.
type Client struct {
	connStr *nsqlitedsn.ConnStr
	httpc   *http.Client
	vdlc    *vdl.Client
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client) error

// WithHTTPTimeout sets the timeout for the default NSQLite HTTP client. Default is 30 seconds.
func WithHTTPTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) error {
		c.httpc.Timeout = timeout
		return nil
	}
}

// WithHTTPTransport sets the transport for the default NSQLite HTTP client. The default is
// http.DefaultTransport with MaxIdleConns, MaxConnsPerHost, and MaxIdleConnsPerHost set to 100.
func WithHTTPTransport(transport *http.Transport) ClientOption {
	return func(c *Client) error {
		c.httpc.Transport = transport
		return nil
	}
}

// WithHTTPClient entirely replaces the default NSQLite HTTP client with a custom one.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) error {
		c.httpc = httpClient
		return nil
	}
}

// NewClient creates a new NSQLite client.
func NewClient(connectionString string, options ...ClientOption) (*Client, error) {
	connStr, err := nsqlitedsn.NewConnStrFromText(connectionString)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	var transport *http.Transport
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = defaultTransport.Clone()
	} else {
		transport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 100

	client := &Client{
		connStr: connStr,
		httpc: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}

	for idx, opt := range options {
		if err := opt(client); err != nil {
			return nil, fmt.Errorf("failed to apply option %d: %w", idx+1, err)
		}
	}

	rpcBaseURL, err := connStr.CreateUrlStr("/rpc")
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc base URL: %w", err)
	}

	builder := vdl.NewClient(rpcBaseURL).WithHTTPClient(client.httpc)
	if connStr.AuthToken != "" {
		builder = builder.WithHeader("Authorization", connStr.AuthToken)
	}
	client.vdlc = builder.Build()

	return client, nil
}

// VDL returns the generated VDL client used internally.
func (c *Client) VDL() *vdl.Client {
	return c.vdlc
}

// SendPing sends a request to the server to check if it is alive. Returns an error
// if the server is not alive.
func (c *Client) SendPing(ctx context.Context) error {
	result, err := c.vdlc.RPCs.System().Procs.Health().Execute(ctx, vdl.Void{})
	if err != nil {
		return err
	}
	if !result.Healthy || !result.Database {
		if result.Message != "" {
			return fmt.Errorf("health check failed: %s", result.Message)
		}
		return fmt.Errorf("health check failed")
	}
	return nil
}

// IsHealthy checks if the server is alive. Returns an error if the server is not healthy.
func (c *Client) IsHealthy(ctx context.Context) error {
	return c.SendPing(ctx)
}

// GetVersion returns the version of the NSQLite server.
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	result, err := c.vdlc.RPCs.System().Procs.Status().Execute(ctx, vdl.Void{})
	if err != nil {
		return "", err
	}
	return result.Version, nil
}

// GetStats returns the database stats from the server.
func (c *Client) GetStats(ctx context.Context) (Stats, error) {
	result, err := c.vdlc.RPCs.System().Procs.Status().Execute(ctx, vdl.Void{})
	if err != nil {
		return Stats{}, err
	}
	return result.Stats, nil
}

// GetSession returns the authentication role for the current session.
func (c *Client) GetSession(ctx context.Context) (SystemSessionOutput, error) {
	return c.vdlc.RPCs.System().Procs.Session().Execute(ctx, vdl.Void{})
}

// SendQueries sends one or more queries to the remote server and returns the responses in same order.
func (c *Client) SendQueries(ctx context.Context, queries []Query) ([]QueryResult, error) {
	result, err := c.vdlc.RPCs.Database().
		Procs.Query().
		Execute(ctx, vdl.DatabaseQueryInput{Queries: queries})
	if err != nil {
		return nil, err
	}
	if len(result.Results) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	return result.Results, nil
}

// SendQuery sends a single query to the remote server and returns the response.
func (c *Client) SendQuery(ctx context.Context, query Query) (QueryResult, error) {
	results, err := c.SendQueries(ctx, []Query{query})
	if err != nil {
		return QueryResult{}, err
	}
	return results[0], nil
}
