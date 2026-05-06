package nsqlitehttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/varavelio/nsqlitego/nsqlitedsn"
)

// Client is an HTTP client for the NSQLite server.
type Client struct {
	connStr *nsqlitedsn.ConnStr
	httpc   *http.Client
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
		transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		}
	}
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 100

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	client := &Client{
		connStr: connStr,
		httpc:   httpClient,
	}

	for idx, opt := range options {
		if err := opt(client); err != nil {
			return nil, fmt.Errorf("failed to apply option %d: %w", idx+1, err)
		}
	}

	return client, nil
}

// newRequest creates a new HTTP request with the NSQLite URL and authentication.
func (c *Client) newRequest(
	ctx context.Context,
	method, path string,
	body io.Reader,
) (*http.Request, error) {
	url, err := c.connStr.CreateUrlStr(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create URL: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	if c.connStr.AuthToken != "" {
		request.Header.Set("Authorization", c.connStr.AuthToken)
	}

	return request, nil
}

// SendPing sends a request to the server to check if it is alive. Returns an error
// if the server is not alive.
func (c *Client) SendPing(ctx context.Context) error {
	request, err := c.newRequest(ctx, http.MethodGet, "/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	response, err := c.httpc.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, response.Body)
		return fmt.Errorf("unwanted response status %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	bodyStr := string(body)

	if strings.ToLower(bodyStr) != "ok" {
		if len(bodyStr) > 100 {
			bodyStr = bodyStr[:100] + "..."
		}
		return fmt.Errorf(
			`health check expected to return "OK" but got "%s"`, bodyStr,
		)
	}

	return nil
}

// IsHealthy checks if the server is alive. Returns an error if the server is
// not healthy.
func (c *Client) IsHealthy(ctx context.Context) error {
	return c.SendPing(ctx)
}

// GetVersion returns the version of the NSQLite server.
func (c *Client) GetVersion(ctx context.Context) (string, error) {
	request, err := c.newRequest(ctx, http.MethodGet, "/version", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	response, err := c.httpc.Do(request)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusUnauthorized {
		_, _ = io.Copy(io.Discard, response.Body)
		return "", fmt.Errorf("authentication failed, please check your credentials")
	}

	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, response.Body)
		return "", fmt.Errorf("unwanted response status: %s", response.Status)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// QueryResponseType is the type of the query response.
type QueryResponseType string

const (
	QueryResponseTypeError    QueryResponseType = "error"
	QueryResponseTypeBegin    QueryResponseType = "begin"
	QueryResponseTypeCommit   QueryResponseType = "commit"
	QueryResponseTypeRollback QueryResponseType = "rollback"
	QueryResponseTypeWrite    QueryResponseType = "write"
	QueryResponseTypeRead     QueryResponseType = "read"
)

// QueryResponse represents the response of a query sent to the remote NSQLite server.
type QueryResponse struct {
	// Type is the type of the query response (error, begin, commit, rollback, write, read)
	//
	//	- Included in all responses.
	Type QueryResponseType `json:"type"`

	// Time is the time taken to execute the query in seconds.
	//
	//	- Included in all responses.
	Time float64 `json:"time"`

	// Error is the error message if the query failed.
	//
	//	- Included in "Type=error" responses.
	Error string `json:"error,omitempty"`

	// TxID is the unique identifier of the new started transaction.
	//
	//	- Included in "Type=begin" responses.
	TxID string `json:"txId,omitempty"`

	// LastInsertID is the last inserted ID of the write query.
	//
	//	- Included in "Type=write" responses.
	LastInsertID int64 `json:"lastInsertId,omitempty"`

	// RowsAffected is the number of rows affected by the write query.
	//
	//	- Included in "Type=write" responses.
	RowsAffected int64 `json:"rowsAffected,omitempty"`

	// Columns is the list of columns returned by the read query.
	//
	//	- Included in "Type=read" responses.
	//	- Included in "Type=write" responses if there are rows in the response.
	Columns []string `json:"columns,omitempty"`

	// Types is the list of types returned by the read query.
	//
	//	- Included in "Type=read" responses.
	//	- Included in "Type=write" responses if there are rows in the response.
	Types []string `json:"types,omitempty"`

	// Rows is the list of rows returned by the read query.
	//
	// Note: Numeric values inside Rows will be of type json.Number to prevent precision loss.
	// You can convert them using val.(json.Number).Int64() or val.(json.Number).Float64().
	//
	//	- Included in "Type=read" responses.
	//	- Included in "Type=write" responses if there are rows in the response.
	Rows [][]any `json:"rows,omitempty"`
}

// QueryParam represents a named (?NNN, :VVV, @VVV, $VVV) or nameless (?) parameter in a SQL query.
type QueryParam struct {
	// Name is the name of the parameter (optional for nameless parameters).
	//
	// Is preferred to include the parameter prefix (e.g. ":id" or "@id") in the name but
	// you can also omit it (e.g. "id").
	Name string `json:"name,omitempty"`
	// Value is the value of the parameter (required).
	Value any `json:"value"`
}

// Query represents the parameters to send a query to the remote server.
type Query struct {
	// Query is the SQL query to send (required).
	Query string `json:"query"`
	// Params are the parameters to send with a parameterized query (optional).
	Params []QueryParam `json:"params,omitempty"`
	// TxID is used to send the query in the context of a transaction (optional).
	TxID string `json:"txId,omitempty"`
}

// SendQueries sends one or more queries to the remote server and returns the responses in same order.
func (c *Client) SendQueries(ctx context.Context, queries []Query) ([]QueryResponse, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(queries); err != nil {
		return nil, fmt.Errorf("failed to encode request body: %w", err)
	}

	request, err := c.newRequest(ctx, http.MethodPost, "/query", &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := c.httpc.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusUnauthorized {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil, fmt.Errorf("authentication failed, please check your credentials")
	}

	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil, fmt.Errorf("unwanted response status: %s", response.Status)
	}

	result := struct {
		Results []QueryResponse `json:"results"`
	}{}

	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	return result.Results, nil
}

// SendQuery sends a single query to the remote server and returns the response.
func (c *Client) SendQuery(ctx context.Context, query Query) (QueryResponse, error) {
	responses, err := c.SendQueries(ctx, []Query{query})
	if err != nil {
		return QueryResponse{}, err
	}

	return responses[0], nil
}

// Stats represents the database stats returned by the server.
type Stats struct {
	// StartedAt is the time the server started.
	StartedAt string `json:"startedAt"`
	// Uptime is the duration of the server uptime.
	Uptime string `json:"uptime"`
	// QueuedBegins is the number of "begin" queries waiting to be executed.
	QueuedBegins int64 `json:"queuedBegins"`
	// QueuedWrites is the number of "write" queries waiting to be executed.
	QueuedWrites int64 `json:"queuedWrites"`
	// QueuedHTTPRequests is the number of HTTP requests waiting to be executed.
	QueuedHTTPRequests int64 `json:"queuedHttpRequests"`
	// Totals is the sum of all stats.
	Totals StatsTotals `json:"totals"`
	// Stats is the breakdown of stats per minute.
	Stats []StatsStat `json:"stats"`
}

type StatsTotals struct {
	// Reads is the number of "read" queries executed.
	Reads int64 `json:"reads"`
	// Writes is the number of "write" queries executed.
	Writes int64 `json:"writes"`
	// Begins is the number of "begin" queries executed.
	Begins int64 `json:"begins"`
	// Commits is the number of "commit" queries executed.
	Commits int64 `json:"commits"`
	// Rollbacks is the number of "rollback" queries executed.
	Rollbacks int64 `json:"rollbacks"`
	// Errors is the number of errors encountered.
	Errors int64 `json:"errors"`
	// HTTPRequests is the number of HTTP requests executed.
	HTTPRequests int64 `json:"httpRequests"`
}

type StatsStat struct {
	// Minute is the minute of the stats.
	Minute string `json:"minute"`
	// Reads is the number of "read" queries executed.
	Reads int64 `json:"reads"`
	// Writes is the number of "write" queries executed.
	Writes int64 `json:"writes"`
	// Begins is the number of "begin" queries executed.
	Begins int64 `json:"begins"`
	// Commits is the number of "commit" queries executed.
	Commits int64 `json:"commits"`
	// Rollbacks is the number of "rollback" queries executed.
	Rollbacks int64 `json:"rollbacks"`
	// Errors is the number of errors encountered.
	Errors int64 `json:"errors"`
	// HTTPRequests is the number of HTTP requests executed.
	HTTPRequests int64 `json:"httpRequests"`
}

// GetStats returns the database stats from the server.
func (c *Client) GetStats(ctx context.Context) (Stats, error) {
	request, err := c.newRequest(ctx, http.MethodGet, "/stats", nil)
	if err != nil {
		return Stats{}, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := c.httpc.Do(request)
	if err != nil {
		return Stats{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusUnauthorized {
		_, _ = io.Copy(io.Discard, response.Body)
		return Stats{}, fmt.Errorf("authentication failed, please check your credentials")
	}

	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, response.Body)
		return Stats{}, fmt.Errorf("unwanted response status: %s", response.Status)
	}

	result := Stats{}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return Stats{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
