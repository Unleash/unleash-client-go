package impactmetrics

import (
	"sort"
	"strings"
	"sync"
)

type MetricLabels map[string]string

type IntMetricSample struct {
	Labels MetricLabels `json:"labels"`
	Value  int64        `json:"value"`
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
	sort.Strings(keys)

	samples := make([]interface{}, 0, len(c.values))
	for _, key := range keys {
		samples = append(samples, IntMetricSample{
			Labels: ParseLabelKey(key),
			Value:  c.values[key],
		})
	}

	c.values = map[string]int64{}

	if len(samples) == 0 {
		samples = append(samples, IntMetricSample{
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

type InMemoryMetricRegistry struct {
	mu       sync.RWMutex
	counters map[string]*counterImpl
}

func NewInMemoryMetricRegistry() *InMemoryMetricRegistry {
	return &InMemoryMetricRegistry{
		counters: map[string]*counterImpl{},
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

func (r *InMemoryMetricRegistry) Collect() []CollectedMetric {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.counters))
	for name := range r.counters {
		names = append(names, name)
	}
	sort.Strings(names)

	var result []CollectedMetric
	for _, name := range names {
		m := r.counters[name].collect()
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
				if ns, ok := s.(IntMetricSample); ok {
					c.Inc(ns.Value, ns.Labels)
				}
			}
		}
	}
}
