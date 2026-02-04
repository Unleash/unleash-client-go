package impactmetrics

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestMetricsAPI(registry *InMemoryMetricRegistry, ctx StaticContext) *MetricsAPI {
	return NewMetricsAPI(registry, ctx, make(chan error, 3))
}

func TestShouldNotRegisterCounterWithEmptyNameOrHelp(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	ctx := StaticContext{AppName: "my-app", Environment: "dev"}
	api := newTestMetricsAPI(registry, ctx)

	api.DefineCounter("some_name", "")
	assert.Nil(t, registry.GetCounter("some_name"))

	api.DefineCounter("", "some_help")
	assert.Nil(t, registry.GetCounter(""))
}

func TestShouldRegisterCounterWithValidNameAndHelp(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	ctx := StaticContext{AppName: "my-app", Environment: "dev"}
	api := newTestMetricsAPI(registry, ctx)

	api.DefineCounter("valid_name", "Valid help text")
	assert.NotNil(t, registry.GetCounter("valid_name"))
}

func TestShouldNotRegisterGaugeWithEmptyNameOrHelp(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	ctx := StaticContext{AppName: "my-app", Environment: "dev"}
	api := newTestMetricsAPI(registry, ctx)

	api.DefineGauge("some_name", "")
	assert.Nil(t, registry.GetGauge("some_name"))

	api.DefineGauge("", "some_help")
	assert.Nil(t, registry.GetGauge(""))
}

func TestShouldRegisterGaugeWithValidNameAndHelp(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	ctx := StaticContext{AppName: "my-app", Environment: "dev"}
	api := newTestMetricsAPI(registry, ctx)

	api.DefineGauge("valid_name", "Valid help text")
	assert.NotNil(t, registry.GetGauge("valid_name"))
}

func TestShouldIncrementCounterWithValidParameters(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	ctx := StaticContext{AppName: "my-app", Environment: "dev"}
	api := newTestMetricsAPI(registry, ctx)

	api.DefineCounter("valid_counter", "help")
	api.IncrementCounter("valid_counter", 5)

	result := registry.Collect()
	metric := findMetric(result, "valid_counter")

	assert.Equal(t, CollectedMetric{
		Name: "valid_counter",
		Help: "help",
		Type: "counter",
		Samples: []interface{}{
			CounterMetricSample{
				Labels: MetricLabels{"appName": "my-app", "environment": "dev"},
				Value:  5,
			},
		},
	}, *metric)
}

func TestShouldIncrementCounterWithDefaultValue(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	ctx := StaticContext{AppName: "my-app", Environment: "dev"}
	api := newTestMetricsAPI(registry, ctx)

	api.DefineCounter("default_counter", "help")
	api.IncrementCounter("default_counter")

	result := registry.Collect()
	metric := findMetric(result, "default_counter")

	assert.Equal(t, CollectedMetric{
		Name: "default_counter",
		Help: "help",
		Type: "counter",
		Samples: []interface{}{
			CounterMetricSample{
				Labels: MetricLabels{"appName": "my-app", "environment": "dev"},
				Value:  1,
			},
		},
	}, *metric)
}

func TestShouldSetGaugeWithValidParameters(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	ctx := StaticContext{AppName: "my-app", Environment: "dev"}
	api := newTestMetricsAPI(registry, ctx)

	api.DefineGauge("valid_gauge", "help")
	api.UpdateGauge("valid_gauge", 10)

	result := registry.Collect()
	metric := findMetric(result, "valid_gauge")

	assert.Equal(t, CollectedMetric{
		Name: "valid_gauge",
		Help: "help",
		Type: "gauge",
		Samples: []interface{}{
			GaugeMetricSample{
				Labels: MetricLabels{"appName": "my-app", "environment": "dev"},
				Value:  10,
			},
		},
	}, *metric)
}

func TestShouldNotRegisterHistogramWithEmptyNameOrHelp(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	ctx := StaticContext{AppName: "my-app", Environment: "dev"}
	api := newTestMetricsAPI(registry, ctx)

	api.DefineHistogram("some_name", "")
	assert.Nil(t, registry.GetHistogram("some_name"))

	api.DefineHistogram("", "some_help")
	assert.Nil(t, registry.GetHistogram(""))
}

func TestShouldRegisterHistogramWithValidNameAndHelp(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	ctx := StaticContext{AppName: "my-app", Environment: "dev"}
	api := newTestMetricsAPI(registry, ctx)

	api.DefineHistogram("valid_name", "Valid help text")
	assert.NotNil(t, registry.GetHistogram("valid_name"))
}

func TestShouldObserveHistogramWithValidParameters(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	ctx := StaticContext{AppName: "my-app", Environment: "dev"}
	api := newTestMetricsAPI(registry, ctx)

	api.DefineHistogram("valid_histogram", "help", 1, 10)
	api.ObserveHistogram("valid_histogram", 1.5)

	result := registry.Collect()
	metric := findMetric(result, "valid_histogram")

	assert.Equal(t, CollectedMetric{
		Name: "valid_histogram",
		Help: "help",
		Type: "histogram",
		Samples: []interface{}{
			HistogramMetricSample{
				Labels: MetricLabels{"appName": "my-app", "environment": "dev"},
				Count:  1,
				Sum:    1.5,
				Buckets: []BucketEntry{
					{Le: 1.0, Count: 0},
					{Le: 10.0, Count: 1},
					{Le: "+Inf", Count: 1},
				},
			},
		},
	}, *metric)
}
