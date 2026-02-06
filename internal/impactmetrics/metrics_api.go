package impactmetrics

import "fmt"

// MetricsAPI provides a high-level interface for recording application metrics.
// It automatically attaches appName and environment labels to all metric operations.
type MetricsAPI struct {
	metricRegistry *InMemoryMetricRegistry
	labels         MetricLabels
	warnings       chan error
}

// NewMetricsAPI creates a new MetricsAPI instance for recording metrics.
// It automatically attaches appName and environment labels to all metric operations.
// Warnings about invalid operations are sent to warningsChannel in a non-blocking manner.
//
// Parameters:
//   - metricRegistry: The underlying metric storage (typically InMemoryMetricRegistry)
//   - context: Static labels (appName, environment) to attach to all metrics
//   - warningsChannel: Buffered channel for receiving validation warnings
func NewMetricsAPI(metricRegistry *InMemoryMetricRegistry, context StaticContext, warningsChannel chan error) *MetricsAPI {
	return &MetricsAPI{
		metricRegistry: metricRegistry,
		labels: MetricLabels{
			"appName":     context.AppName,
			"environment": context.Environment,
		},
		warnings: warningsChannel,
	}
}

// sendWarning sends a warning to the warnings channel in a non-blocking manner.
// If the channel buffer is full, the warning is silently dropped to prevent blocking.
func (api *MetricsAPI) sendWarning(err error) {
	api.warnings <- err
}

// DefineCounter registers a new counter metric with the given name and help text.
// Counter name and help cannot be empty. Invalid definitions are silently ignored with a warning sent to the warnings channel.
func (api *MetricsAPI) DefineCounter(name, help string) {
	if name == "" || help == "" {
		api.sendWarning(fmt.Errorf("counter name or help cannot be empty: name=%s, help=%s", name, help))
		return
	}
	api.metricRegistry.Counter(name, help)
}

// DefineGauge registers a new gauge metric with the given name and help text.
// Gauge name and help cannot be empty. Invalid definitions are silently ignored with a warning sent to the warnings channel.
func (api *MetricsAPI) DefineGauge(name, help string) {
	if name == "" || help == "" {
		api.sendWarning(fmt.Errorf("gauge name or help cannot be empty: name=%s, help=%s", name, help))
		return
	}
	api.metricRegistry.Gauge(name, help)
}

// DefineHistogram registers a new histogram metric with optional custom buckets.
// Histogram name and help cannot be empty. Invalid definitions are silently ignored with a warning sent to the warnings channel.
// If buckets are omitted, default buckets are used.
func (api *MetricsAPI) DefineHistogram(name, help string, buckets ...float64) {
	if name == "" || help == "" {
		api.sendWarning(fmt.Errorf("histogram name or help cannot be empty: name=%s, help=%s", name, help))
		return
	}
	api.metricRegistry.Histogram(name, help, buckets)
}

// IncrementCounter increments the specified counter by 1.
// If the counter is not defined, a warning is sent to the warnings channel and the operation is ignored.
func (api *MetricsAPI) IncrementCounter(name string) {
	api.IncrementCounterBy(name, 1)
}

// IncrementCounterBy increments the specified counter by the given value.
// If the counter is not defined, a warning is sent to the warnings channel and the operation is ignored.
func (api *MetricsAPI) IncrementCounterBy(name string, value int64) {
	counter := api.metricRegistry.GetCounter(name)
	if counter == nil {
		api.sendWarning(fmt.Errorf("counter %q not defined, this counter will not be incremented", name))
		return
	}
	counter.Inc(value, api.labels)
}

// UpdateGauge sets the specified gauge to the given value.
// If the gauge is not defined, a warning is sent to the warnings channel and the operation is ignored.
func (api *MetricsAPI) UpdateGauge(name string, value float64) {
	gauge := api.metricRegistry.GetGauge(name)
	if gauge == nil {
		api.sendWarning(fmt.Errorf("gauge %q not defined, this gauge will not be updated", name))
		return
	}
	gauge.Set(value, api.labels)
}

// ObserveHistogram records the given value in the specified histogram.
// If the histogram is not defined, a warning is sent to the warnings channel and the operation is ignored.
func (api *MetricsAPI) ObserveHistogram(name string, value float64) {
	histogram := api.metricRegistry.GetHistogram(name)
	if histogram == nil {
		api.sendWarning(fmt.Errorf("histogram %q not defined, this histogram will not be updated", name))
		return
	}
	histogram.Observe(value, api.labels)
}
