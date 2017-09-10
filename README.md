# [gentleman](https://github.com/h2non/gentleman)-retry [![Build Status](https://travis-ci.org/h2non/gentleman-retry.png)](https://travis-ci.org/h2non/gentleman-retry) [![GoDoc](https://godoc.org/github.com/h2non/gentleman-retry?status.svg)](https://godoc.org/github.com/h2non/gentleman-retry) [![Coverage Status](https://coveralls.io/repos/github/h2non/gentleman-retry/badge.svg?branch=master)](https://coveralls.io/github/h2non/gentleman-retry?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/h2non/gentleman-retry)](https://goreportcard.com/report/github.com/h2non/gentleman-retry)

[gentleman](https://github.com/h2non/gentleman)'s v2 plugin providing retry policy capabilities to your HTTP clients.

Constant backoff strategy will be used by default with a maximum of 3 attempts, but you use a custom or third-party retry strategies.
Request bodies will be cached in the stack in order to re-send them if needed.

By default, retry will happen in case of network error or server response error (>= 500 || = 429).
You can use a custom `Evaluator` function to determine with custom logic when should retry or not.

Behind the scenes it implements a custom [http.RoundTripper](https://golang.org/pkg/net/http/#RoundTripper)
interface which acts like a proxy to `http.Transport`, in order to take full control of the response and retry the request if needed.

## Installation

```bash
go get -u gopkg.in/h2non/gentleman-retry.v2
```

## Versions

- **[v1](https://github.com/h2non/gentleman-retry/tree/v1)** - First version, uses `gentleman@v1`.
- **[v2](https://github.com/h2non/gentleman-retry/tree/master)** - Latest version, uses `gentleman@v2`.

## API

See [godoc reference](https://godoc.org/github.com/h2non/gentleman-retry) for detailed API documentation.

## Examples

#### Default retry strategy

```go
package main

import (
  "fmt"

  "gopkg.in/h2non/gentleman.v2"
  "gopkg.in/h2non/gentleman-retry.v2"
)

func main() {
  // Create a new client
  cli := gentleman.New()

  // Define base URL
  cli.URL("http://httpbin.org")

  // Register the retry plugin, using the built-in constant back off strategy
  cli.Use(retry.New(retry.ConstantBackoff))

  // Create a new request based on the current client
  req := cli.Request()

  // Define the URL path at request level
  req.Path("/status/503")

  // Set a new header field
  req.SetHeader("Client", "gentleman")

  // Perform the request
  res, err := req.Send()
  if err != nil {
    fmt.Printf("Request error: %s\n", err)
    return
  }
  if !res.Ok {
    fmt.Printf("Invalid server response: %d\n", res.StatusCode)
    return
  }
}
```

#### Exponential retry strategy

I would recommend you using [go-resiliency](https://github.com/eapache/go-resiliency/tree/master/retrier) package for custom retry estrategies:
```go
go get -u gopkg.in/eapache/go-resiliency.v1/retrier
```

```go
package main

import (
  "fmt"
  "time"

  "gopkg.in/h2non/gentleman.v2"
  "gopkg.in/h2non/gentleman-retry.v2"
  "gopkg.in/eapache/go-resiliency.v1/retrier"

)

func main() {
  // Create a new client
  cli := gentleman.New()

  // Define base URL
  cli.URL("http://httpbin.org")

  // Register the retry plugin, using a custom exponential retry strategy
  cli.Use(retry.New(retrier.New(retrier.ExponentialBackoff(3, 100*time.Millisecond), nil)))

  // Create a new request based on the current client
  req := cli.Request()

  // Define the URL path at request level
  req.Path("/status/503")

  // Set a new header field
  req.SetHeader("Client", "gentleman")

  // Perform the request
  res, err := req.Send()
  if err != nil {
    fmt.Printf("Request error: %s\n", err)
    return
  }
  if !res.Ok {
    fmt.Printf("Invalid server response: %d\n", res.StatusCode)
    return
  }
}
```

## License

MIT - Tomas Aparicio
