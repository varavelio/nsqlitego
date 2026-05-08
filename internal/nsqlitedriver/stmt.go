package nsqlitedriver

import (
	"context"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/varavelio/nsqlitego/internal/vdl"
	"github.com/varavelio/nsqlitego/nsqlitehttp"
)

var (
	_ driver.Stmt                           = (*Stmt)(nil)
	_ driver.StmtExecContext                = (*Stmt)(nil)
	_ driver.StmtQueryContext               = (*Stmt)(nil)
	_ driver.RowsColumnTypeDatabaseTypeName = (*QueryRows)(nil)
)

// Stmt represents a prepared statement.
type Stmt struct {
	// conn is the connection associated with the statement.
	conn *Conn
	// query is the query string to be executed with NSQLite.
	query string
}

// Close releases resources associated with the statement.
func (s *Stmt) Close() error {
	return nil
}

// NumInput returns the number of placeholder parameters for the statement.
// -1 indicates that the number is unknown or dynamic.
func (s *Stmt) NumInput() int {
	return -1
}

// ExecResult represents the result of a query.
type ExecResult struct {
	lastInsertId int64
	rowsAffected int64
}

// LastInsertId returns the ID of the last inserted row.
func (r *ExecResult) LastInsertId() (int64, error) {
	return r.lastInsertId, nil
}

// RowsAffected returns the number of rows affected by the query.
func (r *ExecResult) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

// ExecContext executes a query without returning rows (e.g., INSERT, UPDATE).
func (s *Stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	params, err := convertNamedValueToQueryParam(args)
	if err != nil {
		return nil, fmt.Errorf("failed to convert query params: %w", err)
	}
	resp, err := s.conn.client.SendQuery(ctx, nsqlitehttp.Query{
		Query:  s.query,
		Params: &params,
		TxId:   optionalString(s.conn.txID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	if resp.GetError() != "" {
		return nil, fmt.Errorf("failed to execute query: %s", resp.GetError())
	}
	return &ExecResult{
		lastInsertId: resp.GetLastInsertId(),
		rowsAffected: resp.GetRowsAffected(),
	}, nil
}

// Exec executes a query without returning rows (e.g., INSERT, UPDATE, DELETE).
func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), convertValueToNamedValue(args))
}

// QueryRows represents a set of query results.
type QueryRows struct {
	columns []string
	types   []string
	rows    [][]nsqlitehttp.SqliteValue
	rowsLen int
	rowIdx  int
}

// Columns returns the column names.
func (r *QueryRows) Columns() []string {
	return r.columns
}

// Close releases resources associated with the rows.
func (r *QueryRows) Close() error {
	return nil
}

// Next prepares the next row for reading.
func (r *QueryRows) Next(dest []driver.Value) error {
	if r.rowIdx >= r.rowsLen {
		return io.EOF
	}

	row := r.rows[r.rowIdx]
	for i, val := range row {
		dest[i] = sqliteValueToDriverValue(val)
	}

	r.rowIdx++
	return nil
}

// ColumnTypeDatabaseTypeName returns the database type name for the column.
func (r *QueryRows) ColumnTypeDatabaseTypeName(index int) string {
	if index < 0 {
		return ""
	}
	if index >= len(r.types) {
		return ""
	}
	return strings.ToUpper(r.types[index])
}

// QueryContext executes a query that returns rows (e.g., SELECT).
func (s *Stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	params, err := convertNamedValueToQueryParam(args)
	if err != nil {
		return nil, fmt.Errorf("failed to convert query params: %w", err)
	}
	resp, err := s.conn.client.SendQuery(ctx, nsqlitehttp.Query{
		Query:  s.query,
		Params: &params,
		TxId:   optionalString(s.conn.txID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	if resp.GetError() != "" {
		return nil, fmt.Errorf("failed to execute query: %s", resp.GetError())
	}
	types := resp.GetTypes()
	columnTypes := make([]string, len(types))
	for i, value := range types {
		columnTypes[i] = value.String()
	}
	return &QueryRows{
		columns: resp.GetColumns(),
		types:   columnTypes,
		rows:    resp.GetRows(),
		rowsLen: len(resp.GetRows()),
		rowIdx:  0,
	}, nil
}

// Query executes a query that returns rows (e.g., SELECT).
func (s *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), convertValueToNamedValue(args))
}

// convertNamedValueToQueryParam converts driver.NamedValue arguments to []nsqlitehttp.QueryParam.
func convertNamedValueToQueryParam(args []driver.NamedValue) ([]nsqlitehttp.QueryParam, error) {
	converted := make([]nsqlitehttp.QueryParam, len(args))
	for i, arg := range args {
		value, err := driverValueToSqliteValue(arg.Value)
		if err != nil {
			return nil, fmt.Errorf("arg %d: %w", i, err)
		}
		converted[i] = nsqlitehttp.QueryParam{
			Name:  optionalString(arg.Name),
			Value: value,
		}
	}
	return converted, nil
}

func driverValueToSqliteValue(value any) (nsqlitehttp.SqliteValue, error) {
	switch typed := value.(type) {
	case nil:
		return nsqlitehttp.SqliteValue{Null: vdl.Ptr(true)}, nil
	case bool:
		if typed {
			return nsqlitehttp.SqliteValue{Integer: vdl.Ptr(int64(1))}, nil
		}
		return nsqlitehttp.SqliteValue{Integer: vdl.Ptr(int64(0))}, nil
	case int:
		return nsqlitehttp.SqliteValue{Integer: vdl.Ptr(int64(typed))}, nil
	case int8:
		return nsqlitehttp.SqliteValue{Integer: vdl.Ptr(int64(typed))}, nil
	case int16:
		return nsqlitehttp.SqliteValue{Integer: vdl.Ptr(int64(typed))}, nil
	case int32:
		return nsqlitehttp.SqliteValue{Integer: vdl.Ptr(int64(typed))}, nil
	case int64:
		return nsqlitehttp.SqliteValue{Integer: vdl.Ptr(typed)}, nil
	case uint:
		return nsqlitehttp.SqliteValue{Integer: vdl.Ptr(int64(typed))}, nil
	case uint8:
		return nsqlitehttp.SqliteValue{Integer: vdl.Ptr(int64(typed))}, nil
	case uint16:
		return nsqlitehttp.SqliteValue{Integer: vdl.Ptr(int64(typed))}, nil
	case uint32:
		return nsqlitehttp.SqliteValue{Integer: vdl.Ptr(int64(typed))}, nil
	case float32:
		return nsqlitehttp.SqliteValue{Real: vdl.Ptr(float64(typed))}, nil
	case float64:
		return nsqlitehttp.SqliteValue{Real: vdl.Ptr(typed)}, nil
	case string:
		return nsqlitehttp.SqliteValue{Text: vdl.Ptr(typed)}, nil
	case []byte:
		encoded := base64.StdEncoding.EncodeToString(typed)
		return nsqlitehttp.SqliteValue{Blob: vdl.Ptr(encoded)}, nil
	case time.Time:
		formatted := typed.UTC().Format(time.RFC3339Nano)
		return nsqlitehttp.SqliteValue{Text: vdl.Ptr(formatted)}, nil
	default:
		return nsqlitehttp.SqliteValue{}, fmt.Errorf("unsupported value type %T", value)
	}
}

func sqliteValueToDriverValue(value nsqlitehttp.SqliteValue) driver.Value {
	switch {
	case value.Null != nil && *value.Null:
		return nil
	case value.Integer != nil:
		return *value.Integer
	case value.Real != nil:
		return *value.Real
	case value.Text != nil:
		return *value.Text
	case value.Blob != nil:
		decoded, err := base64.StdEncoding.DecodeString(*value.Blob)
		if err == nil {
			return decoded
		}
		return *value.Blob
	default:
		return nil
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return vdl.Ptr(value)
}

// convertValueToNamedValue converts driver.Value arguments to
// []driver.NamedValue.
func convertValueToNamedValue(args []driver.Value) []driver.NamedValue {
	if len(args) == 0 {
		return nil
	}
	namedArgs := make([]driver.NamedValue, len(args))
	for i, arg := range args {
		namedArgs[i] = driver.NamedValue{
			Ordinal: i + 1,
			Value:   arg,
		}
	}
	return namedArgs
}
