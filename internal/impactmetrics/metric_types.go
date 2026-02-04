package impactmetrics

import (
	"math"
	"sort"
	"strings"
	"sync"
)

func isInvalidValue(v float64) bool {
	return math.IsNaN(v) || math.IsInf(v, 0)
}

type MetricLabels map[string]string

type CounterMetricSample struct {
	Labels MetricLabels `json:"labels"`
	Value  int64        `json:"value"`
}

type GaugeMetricSample struct {
	Labels MetricLabels `json:"labels"`
	Value  float64      `json:"value"`
}

type CollectedMetric struct {
	Name    string        `json:"name"`
	Help    string        `json:"help"`
	Type    string        `json:"type"`
	Samples []interface{} `json:"samples"`
}

type ImpactMetricsDataSource interface {
	Collect() []CollectedMetric
	Restore(metrics []CollectedMetric)
}

func LabelKey(labels MetricLabels) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	// Sort keys for deterministic output since Go map iteration order is random - this is a detail to make testing easier and isn't a requirement for the logic to be correct
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + labels[k]
	}
	return strings.Join(parts, ",")
}

func ParseLabelKey(key string) MetricLabels {
	labels := MetricLabels{}
	if key == "" {
		return labels
	}
	for _, pair := range strings.Split(key, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			labels[kv[0]] = kv[1]
		}
	}
	return labels
}

type Counter interface {
	Inc(value int64, labels MetricLabels)
}

type counterImpl struct {
	mu     sync.Mutex
	name   string
	help   string
	values map[string]int64
}

func newCounter(name, help string) *counterImpl {
	return &counterImpl{
		name:   name,
		help:   help,
		values: map[string]int64{},
	}
}

func (c *counterImpl) Inc(value int64, labels MetricLabels) {
	if value <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	key := LabelKey(labels)
	c.values[key] += value
}

func (c *counterImpl) collect() CollectedMetric {
	c.mu.Lock()
	defer c.mu.Unlock()

	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	// Sort keys for deterministic output since Go map iteration order is random
	sort.Strings(keys)

	samples := make([]interface{}, 0, len(c.values))
	for _, key := range keys {
		samples = append(samples, CounterMetricSample{
			Labels: ParseLabelKey(key),
			Value:  c.values[key],
		})
	}

	c.values = map[string]int64{}

	if len(samples) == 0 {
		samples = append(samples, CounterMetricSample{
			Labels: MetricLabels{},
			Value:  0,
		})
	}

	return CollectedMetric{
		Name:    c.name,
		Help:    c.help,
		Type:    "counter",
		Samples: samples,
	}
}

type Gauge interface {
	Inc(value float64, labels MetricLabels)
	Dec(value float64, labels MetricLabels)
	Set(value float64, labels MetricLabels)
}

type gaugeImpl struct {
	mu     sync.Mutex
	name   string
	help   string
	values map[string]float64
}

func newGauge(name, help string) *gaugeImpl {
	return &gaugeImpl{
		name:   name,
		help:   help,
		values: map[string]float64{},
	}
}

func (g *gaugeImpl) Inc(value float64, labels MetricLabels) {
	if isInvalidValue(value) {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	key := LabelKey(labels)
	g.values[key] += value
}

func (g *gaugeImpl) Dec(value float64, labels MetricLabels) {
	if isInvalidValue(value) {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	key := LabelKey(labels)
	g.values[key] -= value
}

func (g *gaugeImpl) Set(value float64, labels MetricLabels) {
	if isInvalidValue(value) {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	key := LabelKey(labels)
	g.values[key] = value
}

func (g *gaugeImpl) collect() CollectedMetric {
	g.mu.Lock()
	defer g.mu.Unlock()

	keys := make([]string, 0, len(g.values))
	for k := range g.values {
		keys = append(keys, k)
	}
	// Sort keys for deterministic output since Go map iteration order is random
	sort.Strings(keys)

	samples := make([]interface{}, 0, len(g.values))
	for _, key := range keys {
		samples = append(samples, GaugeMetricSample{
			Labels: ParseLabelKey(key),
			Value:  g.values[key],
		})
	}

	g.values = map[string]float64{}

	return CollectedMetric{
		Name:    g.name,
		Help:    g.help,
		Type:    "gauge",
		Samples: samples,
	}
}

type InMemoryMetricRegistry struct {
	mu       sync.RWMutex
	counters map[string]*counterImpl
	gauges   map[string]*gaugeImpl
}

func NewInMemoryMetricRegistry() *InMemoryMetricRegistry {
	return &InMemoryMetricRegistry{
		counters: map[string]*counterImpl{},
		gauges:   map[string]*gaugeImpl{},
	}
}

func (r *InMemoryMetricRegistry) Counter(name, help string) Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, exists := r.counters[name]; exists {
		return c
	}
	c := newCounter(name, help)
	r.counters[name] = c
	return c
}

func (r *InMemoryMetricRegistry) GetCounter(name string) Counter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c, exists := r.counters[name]; exists {
		return c
	}
	return nil
}

func (r *InMemoryMetricRegistry) Gauge(name, help string) Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	if g, exists := r.gauges[name]; exists {
		return g
	}
	g := newGauge(name, help)
	r.gauges[name] = g
	return g
}

func (r *InMemoryMetricRegistry) GetGauge(name string) Gauge {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if g, exists := r.gauges[name]; exists {
		return g
	}
	return nil
}

func (r *InMemoryMetricRegistry) Collect() []CollectedMetric {
	r.mu.Lock()
	defer r.mu.Unlock()

	names := make([]string, 0, len(r.counters)+len(r.gauges))
	for name := range r.counters {
		names = append(names, name)
	}
	// Sort names for deterministic output since Go map iteration order is random
	sort.Strings(names)

	var result []CollectedMetric
	for _, name := range names {
		m := r.counters[name].collect()
		if len(m.Samples) > 0 {
			result = append(result, m)
		}
	}

	gaugeNames := make([]string, 0, len(r.gauges))
	for name := range r.gauges {
		gaugeNames = append(gaugeNames, name)
	}
	// Sort names for deterministic output since Go map iteration order is random
	sort.Strings(gaugeNames)

	for _, name := range gaugeNames {
		m := r.gauges[name].collect()
		if len(m.Samples) > 0 {
			result = append(result, m)
		}
	}

	if len(result) == 0 {
		return []CollectedMetric{}
	}
	return result
}

func (r *InMemoryMetricRegistry) Restore(metrics []CollectedMetric) {
	for _, m := range metrics {
		switch m.Type {
		case "counter":
			c := r.Counter(m.Name, m.Help)
			for _, s := range m.Samples {
				if ns, ok := s.(CounterMetricSample); ok {
					c.Inc(ns.Value, ns.Labels)
				}
			}
		case "gauge":
			g := r.Gauge(m.Name, m.Help)
			for _, s := range m.Samples {
				if ns, ok := s.(GaugeMetricSample); ok {
					g.Set(ns.Value, ns.Labels)
				}
			}
		}
	}
}
