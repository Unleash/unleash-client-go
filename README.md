[![Build Status](https://github.com/Unleash/unleash-go-sdk/actions/workflows/build.yml/badge.svg)](https://github.com/Unleash/unleash-go-sdk/actions/workflows/build.yml) [![GoDoc](https://pkg.go.dev/badge/github.com/Unleash/unleash-go-sdk/v6?status.svg)](https://pkg.go.dev/github.com/Unleash/unleash-go-sdk/v6) [![Go Report Card](https://goreportcard.com/badge/github.com/Unleash/unleash-go-sdk/v6)](https://goreportcard.com/report/github.com/Unleash/unleash-go-sdk/v6)
[![Coverage Status](https://coveralls.io/repos/github/Unleash/unleash-go-sdk/badge.svg?branch=v6)](https://coveralls.io/github/Unleash/unleash-go-sdk?branch=v6)

# unleash-go-sdk

The official Unleash SDK for Go. This SDK lets you evaluate feature flags in your Go services and applications.

[Unleash](https://github.com/Unleash/unleash) is an open-source feature management platform. You can use this SDK with [Unleash Enterprise](https://www.getunleash.io/pricing) or [Unleash Open Source](https://github.com/Unleash/unleash).

For complete documentation, see the [Go SDK reference](https://docs.getunleash.io/sdks/go).

## Requirements

- Go 1.21 or later (tested against Go 1.21.x through 1.26.x)

## Installation

Install the SDK module with `go get`:

```bash
go get github.com/Unleash/unleash-go-sdk/v6@latest
```

## Quick start

The following example initializes the SDK and checks a feature flag:

```go
package main

import (
	"net/http"

	unleash "github.com/Unleash/unleash-go-sdk/v6"
)

func main() {
	err := unleash.Initialize(
		unleash.WithAppName("my-go-service"),
		unleash.WithUrl("https://<your-unleash-instance>/api/"),
		unleash.WithCustomHeaders(http.Header{"Authorization": {"<your-backend-token>"}}),
	)

	if err != nil {
		panic(err)
	}

	defer unleash.Close()

	if unleash.IsEnabled("my-feature", unleash.FeatureOptions{}) {
		// new behavior
	}
}
```

## Contributing

### Local development

Clone the repository and the [client specification](https://github.com/Unleash/client-specification) test data. The client specification defines a shared contract that all Unleash SDKs test against.

```bash
git clone https://github.com/Unleash/unleash-go-sdk.git
cd unleash-go-sdk
mkdir -p testdata
git clone https://github.com/Unleash/client-specification.git testdata/client-specification
```

To test a local application against your development copy of the SDK, add a `replace` directive to the application's `go.mod`:

```
replace github.com/Unleash/unleash-go-sdk/v6 => ../unleash-go-sdk/
```

### Running tests

Use `make` to run the test suite:

```bash
make          # vet + tests
make test-race # tests with the race detector
```

### Benchmarking

Run the feature flag evaluation benchmark:

```bash
go test -run=^$ -bench=BenchmarkFeatureToggleEvaluation -benchtime=10s
```

Example output on a MacBook Pro (M1 Pro, 2021) with 16 GB RAM:

```
goos: darwin
goarch: arm64
pkg: github.com/Unleash/unleash-go-sdk/v6
BenchmarkFeatureToggleEvaluation-8 Final Estimated Operations Per Day: 101.131 billion (1.011315e+11)
13635154 854.3 ns/op
PASS
ok github.com/Unleash/unleash-go-sdk/v6 13.388s
```

**854.3 ns/op** translates to roughly **101 billion** evaluations per day on a single CPU core.

### Code style and formatting

Use the following `make` targets to format and lint your code:

```bash
make fmt          # format code
make fmt-check    # check formatting without writing
make strict-check # vet + golint
```

### Releasing

1. Update `clientVersion` in `client.go`.
2. Tag the repository with the new version.
3. Create the release in GitHub.

## License

Apache-2.0
