# Impact Metrics Implementation Guide

## Overview

Impact Metrics are lightweight, application-level time-series metrics that can be recorded alongside feature flag usage and sent to the Unleash server. This document describes the complete implementation in unleash-go-sdk v5 for reference when porting to v6.

## Architecture

```
User Code (Client)
    ↓
MetricsAPI (public interface)
    ↓
InMemoryMetricRegistry (storage)
    ↓
Metric Types: Counter, Gauge, Histogram
    ↓
MetricsData.ImpactMetrics (JSON payload)
    ↓
Unleash Server
```

## Components

### 1. Metric Types (`internal/impactmetrics/metric_types.go`)

**Core Types:**
- `Counter` - monotonic int64 value (only increases)
- `Gauge` - mutable float64 value
- `Histogram` - distribution with buckets

**Key Implementation Details:**

```go
// MetricLabels - map of string key-value pairs
type MetricLabels map[string]string

// Counters: Only accept positive values (value > 0)
func (c *counterImpl) Inc(value int64, labels MetricLabels) {
    if value <= 0 { return }
    // ...
}

// Gauges: Reject NaN and Infinity values
func isInvalidValue(v float64) bool {
    return math.IsNaN(v) || math.IsInf(v, 0)
}

// Histograms: Custom bucket support with automatic +Inf bucket
// Custom marshaling for infinity: math.Inf(1) → "+Inf" string
func (be BucketEntry) MarshalJSON() ([]byte, error) {
    var leValue interface{}
    if math.IsInf(be.Le, 1) {
        leValue = "+Inf"
    } else {
        leValue = be.Le
    }
    return json.Marshal(struct {
        Le    interface{} `json:"le"`
        Count int64       `json:"count"`
    }{...})
}
```

**InMemoryMetricRegistry Pattern:**
- Thread-safe with sync.Mutex/sync.RWMutex
- Separate maps for counters, gauges, histograms
- `Collect()` returns all metrics and clears internal state
- `Restore()` re-inserts metrics (for failed sends)
- Label keys are stringified and sorted for deterministic output

### 2. MetricsAPI (`internal/impactmetrics/metrics_api.go`)

**Purpose:** High-level API for recording metrics with automatic label attachment

**Key Features:**

```go
type MetricsAPI struct {
    metricRegistry *InMemoryMetricRegistry
    labels         MetricLabels  // Pre-allocated appName + environment
    warnings       chan error    // Non-blocking warning channel
}

// Public Methods:
// - DefineCounter(name, help string)
// - DefineGauge(name, help string)
// - DefineHistogram(name, help string, buckets ...float64)
// - IncrementCounter(name string)
// - IncrementCounterBy(name string, value int64)
// - UpdateGauge(name string, value float64)
// - ObserveHistogram(name string, value float64)
```

**Design Patterns:**

1. **Validation on Define:** Empty name/help rejected silently with warning
2. **Operation on Undefined:** Returns warning, operation ignored
3. **Non-blocking Warnings:** Channel sends never block (blocking on a closed channel would panic)
4. **Label Pre-allocation:** Labels map created once at construction

### 3. Client Integration (`client.go`)

**Initialization Pattern:**

```go
// In NewClient()
impactMetricsOptions := impactMetricsOptions{
    appName:     appName,
    environment: environment,
}
impactMetricsChannels := impactMetricsChannels{
    warnings: make(chan error, 3),
}
metricsAPI, metricRegistry := newImpactMetrics(impactMetricsOptions, impactMetricsChannels)

// Pass metricRegistry to metrics system BEFORE starting goroutines
metricsOptions := metricsOptions{
    metricRegistry: metricRegistry,
    // ... other options
}
m := newMetrics(metricsOptions, metricsChannels)
```

**Critical: Pass metricRegistry in Options Constructor**
- Prevents data races (metricRegistry accessed by both main and metrics goroutine)
- Must be set BEFORE `go m.sync()` starts
- No need for nil checks or mutex protection

**Public API:**

```go
func (c *Client) ImpactMetrics() *impactmetrics.MetricsAPI {
    return c.metricsAPI
}
```

### 4. Metrics Payload Integration (`metrics.go`)

**Data Collection:**

```go
func (m *metrics) sendMetrics() {
    m.bucketMu.Lock()
    bucket := m.resetBucket()
    m.bucketMu.Unlock()

    // Collect and clear metrics
    collectedMetrics := impactmetrics.CollectedMetrics(m.metricRegistry.Collect())

    if bucket.IsEmpty() && collectedMetrics.IsEmpty() {
        return
    }

    payload := MetricsData{
        // ... standard fields
        ImpactMetrics: collectedMetrics,
    }
    // ...
}
```

**Key Patterns:**

- `CollectedMetrics` is a type alias with `IsEmpty()` method for consistency
- `Collect()` is called ONCE per metrics interval (not cached)
- Failed sends trigger `Restore()` to re-add metrics to registry
- No persistent state in metrics struct (unlike bucket which is persistent)

### 5. Type Aliases for Consistency

```go
type CollectedMetrics []CollectedMetric

func (cm CollectedMetrics) IsEmpty() bool {
    return len(cm) == 0
}
```

Used for:
- Consistent `.IsEmpty()` calls (bucket has `IsEmpty()` method)
- Better readability in sendMetrics()

## Test Patterns

### Unit Tests for Metrics

**Use Real Objects, Not Mocks:**
```go
func TestShouldIncrementCounterWithValidParameters(t *testing.T) {
    registry := NewInMemoryMetricRegistry()
    ctx := StaticContext{AppName: "my-app", Environment: "dev"}
    api := newTestMetricsAPI(registry, ctx)

    api.DefineCounter("valid_counter", "help")
    api.IncrementCounterBy("valid_counter", 5)

    result := registry.Collect()
    metric := findMetric(result, "valid_counter")

    assert.Equal(t, CollectedMetric{...}, *metric)
}
```

**Error Path Testing:**
```go
func TestShouldSendWarningWhenIncrementingUndefinedCounter(t *testing.T) {
    registry := NewInMemoryMetricRegistry()
    warnings := make(chan error, 3)
    api := NewMetricsAPI(registry, ctx, warnings)

    api.IncrementCounter("undefined")

    select {
    case err := <-warnings:
        assert.Contains(t, err.Error(), "counter")
        assert.Contains(t, err.Error(), "not defined")
    case <-time.After(100*time.Millisecond):
        t.Fatal("expected warning not sent")
    }
}
```

### Integration Tests

**Channel-Based Pattern (No time.Sleep):**
```go
func TestImpactMetricsSentInPayload(t *testing.T) {
    payloads := make(chan []byte, 1)

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "POST" && r.URL.Path == "/client/metrics" {
            body, _ := io.ReadAll(r.Body)
            payloads <- body
            w.WriteHeader(http.StatusAccepted)
        }
    }))

    client, _ := NewClient(
        WithUrl(server.URL),
        WithMetricsInterval(5*time.Millisecond),  // Fast interval for tests
    )
    defer client.Close()

    api := client.ImpactMetrics()
    api.DefineCounter("purchases", "Number of purchases")
    api.IncrementCounter("purchases")

    payloadBytes := <-payloads  // Block until metrics sent

    // Assert with JSONEq for full payload comparison
    assert.JSONEq(t, expectedJSON, string(payloadBytes))
}
```

**Test Failure Scenario:**
```go
func TestImpactMetricsResentAfterFailure(t *testing.T) {
    payloads := make(chan []byte, 2)
    var failNext atomic.Bool
    failNext.Store(true)

    // First POST returns 500, second returns 202
    // Metrics should be restored and re-sent

    <-payloads                  // First attempt (fails)
    secondPayload := <-payloads // Second attempt (succeeds)
}
```

## JSON Serialization

**MetricsData Structure:**
```json
{
  "appName": "test-app",
  "bucket": { ... },
  "impactMetrics": [
    {
      "name": "purchases",
      "help": "Number of purchases",
      "type": "counter",
      "samples": [
        {
          "labels": {"appName": "test-app", "environment": "test"},
          "value": 1
        }
      ]
    }
  ]
}
```

**Special JSON Handling:**
- Histogram bucket edges: `math.Inf(1)` → `"+Inf"` (custom MarshalJSON)
- Gauges: float64 values (supports decimals, not integers)
- Counters: int64 values (always positive)

## Important Decisions

### 1. Pre-allocate Labels at Construction
- Efficiency: Labels map created once
- Simplicity: No need to recalculate label strings on each operation
- Trade-off: LabelKey string generation still allocates slices (minor impact)

### 2. Separate Counter/Gauge Methods
- `IncrementCounter()` and `IncrementCounterBy()` instead of variadic
- Avoids confusion with optional parameters
- Clear semantics for default value (1)

### 3. No Persistent Metrics State in metrics struct
- Metrics cleared after each `Collect()`
- Unlike bucket which is persistent/cumulative
- Allows proper Restore() handling on failed sends

### 4. Type Alias for Consistency
- `CollectedMetrics` alias enables `.IsEmpty()` method
- Matches bucket pattern for idiomatic Go
- Improves readability in sendMetrics()

### 5. Non-blocking Warning Channel
- Warnings never block the application
- Silent drop if buffer fills (capacity 3)
- Matches pattern used in delta_processor.go

### 6. Validation Strategy
- Empty name/help rejected on Define*
- Undefined metrics rejected on operation (with warning)
- Invalid values (NaN, Inf, negative) silently dropped
- No exceptions/panics for user errors

## Race Condition Prevention

**Critical: Pass metricRegistry in Constructor Options**

```go
// ✅ CORRECT - No race condition
metricsOptions := metricsOptions{
    metricRegistry: metricRegistry,  // Set BEFORE goroutine starts
}
m := newMetrics(metricsOptions, metricsChannels)
go m.sync()  // Now safe to access m.metricRegistry

// ❌ WRONG - Race condition
m := newMetrics(metricsOptions, metricsChannels)
go m.sync()  // Goroutine may see nil metricRegistry
m.setMetricRegistry(metricRegistry)  // Set AFTER goroutine started
```

**Why it matters:**
- `sendMetrics()` reads metricRegistry in goroutine
- Main thread writes metricRegistry field
- Without proper initialization, data race on metricRegistry field

## Code Quality Standards

### Godoc Comments
All public functions must have godoc:
```go
// NewMetricsAPI creates a new MetricsAPI instance for recording metrics.
// It automatically attaches appName and environment labels to all metric operations.
// Warnings about invalid operations are sent to warningsChannel in a non-blocking manner.
func NewMetricsAPI(...)
```

### Test Coverage
- Unit tests: 39+ tests for metric types and API
- Integration tests: End-to-end payload verification
- Error paths: Undefined metrics, empty names, invalid values
- Race detector: All tests pass with `-race` flag

### Error Handling
- Validation errors → warnings channel (non-blocking)
- No panics or exceptions
- Silent failure for invalid operations
- Clear error messages for debugging

## Performance Characteristics

- **Labels Pre-allocation:** O(1) label attachment per operation
- **LabelKey Generation:** O(n) where n = number of label keys (typically 2-3)
- **Mutex Contention:** Minimal (used only for define and collect)
- **Memory:** One registry instance shared across all metrics
- **Channel Buffering:** 3 capacity for warnings (low memory footprint)

## Migration to v6

When porting to v6:

1. **Copy Core Types:** `metric_types.go` is stable and self-contained
2. **Update MetricsAPI:** Check for any client API changes
3. **Adjust Integration Points:** May change based on v6 client structure
4. **Update Tests:** Pattern remains the same, may need URL/interval adjustments
5. **Review Godoc:** Ensure consistency with v6 documentation style

## Files Summary

| File | Lines | Purpose |
|------|-------|---------|
| `internal/impactmetrics/metric_types.go` | 581 | Core metric implementations |
| `internal/impactmetrics/metric_types_test.go` | 379 | Unit tests for metrics (35+ tests) |
| `internal/impactmetrics/metrics_api.go` | 98 | High-level API for users |
| `internal/impactmetrics/metrics_api_test.go` | 234 | API and integration tests |
| `internal/impactmetrics/context.go` | 22 | Static context extraction |
| `client.go` | Changes | Client integration and ImpactMetrics() method |
| `config.go` | Changes | Options and types for impact metrics |
| `metrics.go` | Changes | Payload collection and Restore logic |
| `impact_metrics_test.go` | 172 | End-to-end integration tests |
| `README.md` | Changes | User documentation |

## Key Test Statistics

- **Total Tests:** 39 unit tests + 2 integration tests
- **Coverage:** 93.8% on metric_types.go
- **Race Detector:** All tests pass with `-race` flag
- **Speed:** Complete suite runs in <1s
- **Failure Scenarios:** Tested with HTTP 500 responses

## Lessons Learned

1. **Always pass mutable state in constructors** - Prevents race conditions with goroutines
2. **Use channels for test synchronization** - Better than time.Sleep()
3. **Type aliases enable consistent patterns** - `.IsEmpty()` works on both bucket and metrics
4. **Custom JSON marshaling is powerful** - Infinity handling in histograms
5. **Non-blocking channel operations** - Use select with default case to prevent deadlocks
6. **Godoc is not optional** - IDE tooltips and godoc.org depend on proper comments
7. **Error paths need testing** - Warnings channel sends must be verified

## References

- Prometheus histogram bucket semantics: https://prometheus.io/docs/concepts/metric_types/
- Go concurrency patterns: https://go.dev/blog/pipelines
- Custom JSON marshaling: https://pkg.go.dev/encoding/json#Marshaler
