# nsqlitehttp

<a href="https://pkg.go.dev/github.com/varavelio/nsqlitego/nsqlitehttp">
  <img src="https://pkg.go.dev/badge/github.com/varavelio/nsqlitego/nsqlitehttp" alt="Go Reference"/>
</a>

An HTTP client package for directly interacting with the **NSQLite database
engine** over HTTP/HTTPS.

Use this if you want a lower-level approach than the
[`database/sql`](https://pkg.go.dev/database/sql) interface provided by
[`nsqlitego`](https://github.com/varavelio/nsqlitego).

## Features

- Issue raw JSON-based requests to an NSQLite server.
- Customizable headers, request methods, and endpoints.
- Handle responses as JSON or plain text.
- Zero dependencies outside the standard library.

## Installation

```bash
go get github.com/varavelio/nsqlitego
```

> **Note**: This package is part of the
> [`nsqlitego`](https://github.com/varavelio/nsqlitego) repository.\
> Import it as:
>
> ```go
> import "github.com/varavelio/nsqlitego/nsqlitehttp"
> ```

## Usage

### Creating a Client

```go
client, err := nsqlitehttp.NewClient("http://localhost:9876?authToken=myToken")
if err != nil {
  panic(err)
}
```

### Sending Queries

```go
resp, err := client.SendQuery(context.TODO(), nsqlitehttp.Query{
  Query:  "SELECT id, name FROM users WHERE id > :id",
  Params: []nsqlitehttp.QueryParam{{Name: "id", Value: 100}},
  // TxId can be optionally set if you are managing transactions at this level
})
if err != nil {
  panic(err)
}

fmt.Printf("Response Type: %s\n", resp.Type)
if resp.Type == nsqlitehttp.QueryResponseRead {
  // Access resp.Columns, resp.Rows, etc.
}
```

You can also send multiple queries in a single request using
`client.SendQueries(ctx, queries)`.

### Ping / Health Check

```go
if err := client.SendPing(context.TODO()); err != nil {
  fmt.Println("Server is not healthy:", err)
} else {
  fmt.Println("Server is up and running.")
}
```

### Advanced Usage

Please refer to the
[nsqlitehttp Go Reference](https://pkg.go.dev/github.com/varavelio/nsqlitego/nsqlitehttp)
for more details.
