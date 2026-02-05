package impactmetrics

import "fmt"

type MetricsAPI struct {
	metricRegistry *InMemoryMetricRegistry
	labels         MetricLabels
	warnings       chan error
}

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

func (api *MetricsAPI) sendWarning(err error) {
	api.warnings <- err
}

func (api *MetricsAPI) DefineCounter(name, help string) {
	if name == "" || help == "" {
		api.sendWarning(fmt.Errorf("counter name or help cannot be empty: name=%s, help=%s", name, help))
		return
	}
	api.metricRegistry.Counter(name, help)
}

func (api *MetricsAPI) DefineGauge(name, help string) {
	if name == "" || help == "" {
		api.sendWarning(fmt.Errorf("gauge name or help cannot be empty: name=%s, help=%s", name, help))
		return
	}
	api.metricRegistry.Gauge(name, help)
}

func (api *MetricsAPI) DefineHistogram(name, help string, buckets ...float64) {
	if name == "" || help == "" {
		api.sendWarning(fmt.Errorf("histogram name or help cannot be empty: name=%s, help=%s", name, help))
		return
	}
	api.metricRegistry.Histogram(name, help, buckets)
}

func (api *MetricsAPI) IncrementCounter(name string) {
	api.IncrementCounterBy(name, 1)
}

func (api *MetricsAPI) IncrementCounterBy(name string, value int64) {
	counter := api.metricRegistry.GetCounter(name)
	if counter == nil {
		api.sendWarning(fmt.Errorf("counter %q not defined, this counter will not be incremented", name))
		return
	}
	counter.Inc(value, api.labels)
}

func (api *MetricsAPI) UpdateGauge(name string, value float64) {
	gauge := api.metricRegistry.GetGauge(name)
	if gauge == nil {
		api.sendWarning(fmt.Errorf("gauge %q not defined, this gauge will not be updated", name))
		return
	}
	gauge.Set(value, api.labels)
}

func (api *MetricsAPI) ObserveHistogram(name string, value float64) {
	histogram := api.metricRegistry.GetHistogram(name)
	if histogram == nil {
		api.sendWarning(fmt.Errorf("histogram %q not defined, this histogram will not be updated", name))
		return
	}
	histogram.Observe(value, api.labels)
}
