[![Build Status](https://travis-ci.org/Unleash/unleash-client-go.svg?branch=master)](https://travis-ci.org/Unleash/unleash-client-go) [![GoDoc](https://godoc.org/github.com/Unleash/unleash-client-go?status.svg)](https://godoc.org/github.com/Unleash/unleash-client-go) [![Go Report Card](https://goreportcard.com/badge/github.com/Unleash/unleash-client-go)](https://goreportcard.com/report/github.com/Unleash/unleash-client-go)

# unleash-client-go
Unleash Client for Go.  Read more about the [Unleash project](https://github.com/finn-no/unleash) 


## Getting started

### 1. Install unleash-client-go

```bash
go get github.com/Unleash/unleash-client-go
```

### 2. Initialize unleash

The easiest way to get started with Unleash is to initialize it early in your application code:

```go
import (
	"github.com/Unleash/unleash-client-go"
)

func init() {
	unleash.Initialize(
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
