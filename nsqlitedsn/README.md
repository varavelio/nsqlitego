# nsqlitedsn

<a href="https://pkg.go.dev/github.com/varavelio/nsqlitego/nsqlitedsn">
  <img src="https://pkg.go.dev/badge/github.com/varavelio/nsqlitego/nsqlitedsn" alt="Go Reference"/>
</a>

A lightweight utility package for parsing and manipulating **NSQLite**
connection strings.

## Features

- Parse URLs of the form `http://host:port?authToken=xyz` (or `https`).
- Cleanly separate protocol, host, port, and optional auth token.
- Provide helper methods for constructing and joining paths.
- Zero dependencies outside the standard library.

## Installation

```bash
go get github.com/varavelio/nsqlitego
```

> **Note**: This package comes bundled with
> [`nsqlitego`](https://github.com/varavelio/nsqlitego).\
> Import it as follows:
>
> ```go
> import "github.com/varavelio/nsqlitego/nsqlitedsn"
> ```

## Usage

### Parsing a Connection String

```go
connStr, err := nsqlitedsn.NewConnStrFromText("https://example.com:9999?authToken=abc123")
if err != nil {
  panic(err)
}

fmt.Println(connStr.Protocol)  // https
fmt.Println(connStr.Host)      // example.com
fmt.Println(connStr.Port)      // 9999
fmt.Println(connStr.AuthToken) // abc123
```

### Constructing URLs

```go
urlStr, err := connStr.CreateUrlStr("/api/v1/query?param=demo")
if err != nil {
  panic(err)
}
fmt.Println(urlStr)
// Output: https://example.com:9999/api/v1/query?param=demo
```
