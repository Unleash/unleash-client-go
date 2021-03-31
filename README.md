[![Build Status](https://github.com/Unleash/unleash-client-go/actions/workflows/build.yml/badge.svg)](https://github.com/Unleash/unleash-client-go/actions/workflows/build.yml) [![GoDoc](https://godoc.org/github.com/Unleash/unleash-client-go?status.svg)](https://godoc.org/github.com/Unleash/unleash-client-go) [![Go Report Card](https://goreportcard.com/badge/github.com/Unleash/unleash-client-go)](https://goreportcard.com/report/github.com/Unleash/unleash-client-go)
[![Coverage Status](https://coveralls.io/repos/github/Unleash/unleash-client-go/badge.svg?branch=v3)](https://coveralls.io/github/Unleash/unleash-client-go?branch=v3)

# unleash-client-go

Unleash Client for Go. Read more about the [Unleash project](https://github.com/Unleash/unleash)

**Version 3.x of the client requires `unleash-server` v3.x or higher.**

## Go Version

The client is currently tested against Go 1.10.x and 1.13.x. These versions will be updated
as new versions of Go are released.

The client may work on older versions of Go as well, but is not actively tested.

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
		unleash.WithCustomHeaders(http.Header{"Authorization": {"<API token>"}}),
	)
}
```
#### Preloading feature toggles

If you'd like to prebake your application with feature toggles (maybe you're working without persistent storage, so Unleash's backup isn't available), you can replace the defaultStorage implementation with a BootstrapStorage. This allows you to pass in a reader to where data in the format of `/api/client/features` can be found.

#### Bootstrapping from file

Bootstrapping from file on disk is then done using something similar to

```go
import (
	"github.com/Unleash/unleash-client-go/v3"
)

func init() {
    myBootstrap := os.Open("bootstrapfile.json") // or wherever your file is located at runtime
    // BootstrapStorage handles the case where Reader is nil
    unleash.Initialize(
	    unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithAppName("my-application"),
		unleash.WithUrl("http://unleash.herokuapp.com/api/"),
        unleash.WithStorage(&BootstrapStorage{Reader: myBootstrap})
	)
}
```

#### Bootstrapping from S3

Bootstrapping from S3 is then done by downloading the file using the aws library and then passing in a Reader to the just downloaded file

```go
import (
	"github.com/Unleash/unleash-client-go/v3"
    "github.com/aws/aws-sdk-go/service/s3"
)

func init() {
    // The session the S3 Downloader will use
    sess := session.Must(session.NewSession())

    // Create a downloader with the session and default options
    downloader := s3manager.NewDownloader(sess)

    // Create a file to write the S3 Object contents to.
    f, err := os.Create("bootstrap.json")
    if err != nil {
        return fmt.Errorf("failed to create file %q, %v", filename, err)
    }

    // Write the contents of S3 Object to the file
    n, err := downloader.Download(f, &s3.GetObjectInput{
        Bucket: aws.String(myBucket),
        Key:    aws.String(myString),
    })
    if err != nil {
        return fmt.Errorf("failed to download file, %v", err)
    }


    // BootstrapStorage handles the case where Reader is nil
    unleash.Initialize(
	    unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithAppName("my-application"),
		unleash.WithUrl("http://unleash.herokuapp.com/api/"),
        unleash.WithStorage(&BootstrapStorage{Reader: f})
	)
}
```

#### Bootstrapping from Google

Since the google cloud storage API returns a Reader, implementing a Bootstrap from GCS is done using something similar to

```go
import (
	"github.com/Unleash/unleash-client-go/v3"
    "cloud.google.com/go/storage"
)

func init() {
    ctx := context.Background() // Configure Google Cloud context
    client, err := storage.NewClient(ctx) // Configure your client
    if err != nil {
        // TODO: Handle error.
    }

    // Fetch the bucket, then object and then create a reader
    reader := client.Bucket(bucketName).Object("my-bootstrap.json").NewReader(ctx)

    // BootstrapStorage handles the case where Reader is nil
    unleash.Initialize(
	    unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithAppName("my-application"),
		unleash.WithUrl("http://unleash.herokuapp.com/api/"),
        unleash.WithStorage(&unleash.BootstrapStorage{Reader: reader})
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
- FlexibleRolloutStrategy
- GradualRolloutUserIdStrategy
- GradualRolloutSessionIdStrategy
- GradualRolloutRandomStrategy
- RemoteAddressStrategy
- ApplicationHostnameStrategy

[Read more about activation strategies in the docs](https://docs.getunleash.io/user_guide/activation_strategy).

### Unleash context

In order to use some of the common activation strategies you must provide an
[unleash-context](https://docs.getunleash.io/user_guide/unleash_context).
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

## Adding client specifications

In order to make sure the unleash clients uphold their contract, we have defined a set of
client specifications that define this contract. These are used to make sure that each unleash client
at any time adhere to the contract, and define a set of functionality that is core to unleash. You can view
the [client specifications here](https://github.com/Unleash/client-specification).

In order to make the tests run please do the following steps.

```
// in repository root
// testdata is gitignored
mkdir testdata
cd testdata
git clone https://github.com/Unleash/client-specification.git
```

Requirements:

- make
- golint (go get -u golang.org/x/lint/golint)

Run tests:

    make

Run lint check:

    make lint

Run code-style checks:(currently failing)

    make strict-check

Run race-tests:

    make test-all
