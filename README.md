[![Build Status](https://travis-ci.org/Unleash/unleash-client-go.svg?branch=master)](https://travis-ci.org/Unleash/unleash-client-go) [![GoDoc](https://godoc.org/github.com/Unleash/unleash-client-go?status.svg)](https://godoc.org/github.com/Unleash/unleash-client-go) [![Go Report Card](https://goreportcard.com/badge/github.com/Unleash/unleash-client-go)](https://goreportcard.com/report/github.com/Unleash/unleash-client-go)

# unleash-client-go
Unleash Client for Go.  Read more about the [Unleash project](https://github.com/Unleash/unleash)

**Version 3.x of the client requires `unleash-server` v3.x or higher.**

## Getting started

### 1. Install unleash-client-go

To install the latest version of the client use:

```bash
go get github.com/Unleash/unleash-client-go/v3
```

If you are still using Unleash Server v2.x.x, then you should use:

```bash
go get github.com/Unleash/unleash-client-go
```

### 2. Initialize unleash

The easiest way to get started with Unleash is to initialize it early in your application code:

```go
import (
	"github.com/Unleash/unleash-client-go/v3"
)

func init() {
	unleash.Initialize(
		unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithAppName("my-application"),
		unleash.WithUrl("http://unleash.herokuapp.com/api/"),
	)
}
```

### 3. Use unleash

After you have initialized the unleash-client you can easily check if a feature
toggle is enabled or not.

```go
unleash.IsEnabled("app.ToggleX")
```

### 4. Stop unleash

To shut down the client (turn off the polling) you can simply call the
destroy-method. This is typically not required.

unleash.Close()

### Built in activation strategies
The Go client comes with implementations for the built-in activation strategies
provided by unleash.

- DefaultStrategy
- UserIdStrategy
- GradualRolloutUserIdStrategy
- GradualRolloutSessionIdStrategy
- GradualRolloutRandomStrategy
- RemoteAddressStrategy
- ApplicationHostnameStrategy

Read more about the strategies in [activation-strategy.md](https://github.com/Unleash/unleash/blob/master/docs/activation-strategies.md).

### Unleash context

In order to use some of the common activation strategies you must provide a
[unleash-context](https://github.com/Unleash/unleash/blob/master/docs/unleash-context.md).
This client SDK allows you to send in the unleash context as part of the `isEnabled` call:

```go
ctx := context.Context{
    UserId: "123",
    SessionId: "some-session-id",
    RemoteAddress: "127.0.0.1",
}

unleash.IsEnabled("someToggle", unleash.WithContext(ctx))
```

### Caveat
This client uses go routines to report several events and doesn't drain the channel by default. So you need to either register a listener using `WithListener` or drain the channel "manually" (demonstrated in [this example](https://github.com/Unleash/unleash-client-go/blob/master/example_with_instance_test.go)).

## Development

Requirements:
* make
* golint (go get -u golang.org/x/lint/golint)

Run tests:

    make 
    
Run lint check:

    make lint
    
Run code-style checks:(currently failing)
    
    make strict-check
    
Run race-tests(currently failing):
 
    make test-all
