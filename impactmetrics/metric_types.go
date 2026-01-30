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

type NumericMetricSample struct {
	Labels MetricLabels `json:"labels"`
	Value  float64      `json:"value"`
}

type CollectedMetric struct {
	Name    string        `json:"name"`
	Help    string        `json:"help"`
	Type    string        `json:"type"`
	Samples []interface{} `json:"samples"`
}

type BucketEntry struct {
	Le    interface{} `json:"le"`
	Count int64       `json:"count"`
}

type BucketMetricSample struct {
	Labels  MetricLabels  `json:"labels"`
	Count   int64         `json:"count"`
	Sum     float64       `json:"sum"`
	Buckets []BucketEntry `json:"buckets"`
}

var DefaultHistogramBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

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
	keys   []string // insertion order
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
	if _, exists := c.values[key]; !exists {
		c.keys = append(c.keys, key)
	}
	c.values[key] += value
}

func (c *counterImpl) collect() CollectedMetric {
	c.mu.Lock()
	defer c.mu.Unlock()

	samples := make([]interface{}, 0, len(c.values))
	for _, key := range c.keys {
		samples = append(samples, NumericMetricSample{
			Labels: ParseLabelKey(key),
			Value:  float64(c.values[key]),
		})
	}

	c.values = map[string]int64{}
	c.keys = nil

	if len(samples) == 0 {
		samples = append(samples, NumericMetricSample{
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
	keys   []string // insertion order
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
	if _, exists := g.values[key]; !exists {
		g.keys = append(g.keys, key)
	}
	g.values[key] += value
}

func (g *gaugeImpl) Dec(value float64, labels MetricLabels) {
	if isInvalidValue(value) {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	key := LabelKey(labels)
	if _, exists := g.values[key]; !exists {
		g.keys = append(g.keys, key)
	}
	g.values[key] -= value
}

func (g *gaugeImpl) Set(value float64, labels MetricLabels) {
	if isInvalidValue(value) {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	key := LabelKey(labels)
	if _, exists := g.values[key]; !exists {
		g.keys = append(g.keys, key)
	}
	g.values[key] = value
}

func (g *gaugeImpl) collect() CollectedMetric {
	g.mu.Lock()
	defer g.mu.Unlock()

	samples := make([]interface{}, 0, len(g.values))
	for _, key := range g.keys {
		samples = append(samples, NumericMetricSample{
			Labels: ParseLabelKey(key),
			Value:  g.values[key],
		})
	}

	g.values = map[string]float64{}
	g.keys = nil

	return CollectedMetric{
		Name:    g.name,
		Help:    g.help,
		Type:    "gauge",
		Samples: samples,
	}
}

type Histogram interface {
	Observe(value float64, labels MetricLabels)
}

type histogramData struct {
	count   int64
	sum     float64
	buckets map[float64]int64
}

type histogramImpl struct {
	mu      sync.Mutex
	name    string
	help    string
	buckets []float64
	values  map[string]*histogramData
	keys    []string // insertion order
}

func newHistogram(name, help string, buckets []float64) *histogramImpl {
	if len(buckets) == 0 {
		buckets = DefaultHistogramBuckets
	}
	seen := map[float64]bool{}
	var filtered []float64
	for _, b := range buckets {
		if b != math.Inf(1) && !seen[b] {
			seen[b] = true
			filtered = append(filtered, b)
		}
	}
	sort.Float64s(filtered)
	filtered = append(filtered, math.Inf(1))

	return &histogramImpl{
		name:    name,
		help:    help,
		buckets: filtered,
		values:  map[string]*histogramData{},
	}
}

func (h *histogramImpl) Observe(value float64, labels MetricLabels) {
	if isInvalidValue(value) {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	key := LabelKey(labels)
	data, exists := h.values[key]
	if !exists {
		data = &histogramData{
			buckets: make(map[float64]int64, len(h.buckets)),
		}
		for _, b := range h.buckets {
			data.buckets[b] = 0
		}
		h.values[key] = data
		h.keys = append(h.keys, key)
	}

	data.count++
	data.sum += value
	for _, b := range h.buckets {
		if value <= b {
			data.buckets[b]++
		}
	}
}

func (h *histogramImpl) restore(sample BucketMetricSample) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := LabelKey(sample.Labels)
	data := &histogramData{
		count:   sample.Count,
		sum:     sample.Sum,
		buckets: make(map[float64]int64, len(sample.Buckets)),
	}
	for _, b := range sample.Buckets {
		le := bucketLeToFloat(b.Le)
		data.buckets[le] = b.Count
	}
	if _, exists := h.values[key]; !exists {
		h.keys = append(h.keys, key)
	}
	h.values[key] = data
}

func bucketLeToFloat(le interface{}) float64 {
	switch v := le.(type) {
	case string:
		if v == "+Inf" {
			return math.Inf(1)
		}
	case float64:
		return v
	}
	return 0
}

func formatLe(b float64) interface{} {
	if math.IsInf(b, 1) {
		return "+Inf"
	}
	return b
}

func (h *histogramImpl) collect() CollectedMetric {
	h.mu.Lock()
	defer h.mu.Unlock()

	samples := make([]interface{}, 0, len(h.values))
	for _, key := range h.keys {
		data := h.values[key]
		bucketEntries := make([]BucketEntry, len(h.buckets))
		for i, b := range h.buckets {
			bucketEntries[i] = BucketEntry{
				Le:    formatLe(b),
				Count: data.buckets[b],
			}
		}
		samples = append(samples, BucketMetricSample{
			Labels:  ParseLabelKey(key),
			Count:   data.count,
			Sum:     data.sum,
			Buckets: bucketEntries,
		})
	}

	h.values = map[string]*histogramData{}
	h.keys = nil

	if len(samples) == 0 {
		bucketEntries := make([]BucketEntry, len(h.buckets))
		for i, b := range h.buckets {
			bucketEntries[i] = BucketEntry{
				Le:    formatLe(b),
				Count: 0,
			}
		}
		samples = append(samples, BucketMetricSample{
			Labels:  MetricLabels{},
			Count:   0,
			Sum:     0,
			Buckets: bucketEntries,
		})
	}

	return CollectedMetric{
		Name:    h.name,
		Help:    h.help,
		Type:    "histogram",
		Samples: samples,
	}
}

type InMemoryMetricRegistry struct {
	mu            sync.RWMutex
	counters      map[string]*counterImpl
	counterKeys   []string // insertion order
	gauges        map[string]*gaugeImpl
	gaugeKeys     []string // insertion order
	histograms    map[string]*histogramImpl
	histogramKeys []string // insertion order
}

func NewInMemoryMetricRegistry() *InMemoryMetricRegistry {
	return &InMemoryMetricRegistry{
		counters:   map[string]*counterImpl{},
		gauges:     map[string]*gaugeImpl{},
		histograms: map[string]*histogramImpl{},
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
	r.counterKeys = append(r.counterKeys, name)
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
	r.gaugeKeys = append(r.gaugeKeys, name)
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

func (r *InMemoryMetricRegistry) Histogram(name, help string, buckets []float64) Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, exists := r.histograms[name]; exists {
		return h
	}
	h := newHistogram(name, help, buckets)
	r.histograms[name] = h
	r.histogramKeys = append(r.histogramKeys, name)
	return h
}

func (r *InMemoryMetricRegistry) GetHistogram(name string) Histogram {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if h, exists := r.histograms[name]; exists {
		return h
	}
	return nil
}

func (r *InMemoryMetricRegistry) Collect() []CollectedMetric {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []CollectedMetric
	for _, name := range r.counterKeys {
		m := r.counters[name].collect()
		if len(m.Samples) > 0 {
			result = append(result, m)
		}
	}
	for _, name := range r.gaugeKeys {
		m := r.gauges[name].collect()
		if len(m.Samples) > 0 {
			result = append(result, m)
		}
	}
	for _, name := range r.histogramKeys {
		m := r.histograms[name].collect()
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
			c, ok := r.Counter(m.Name, m.Help).(*counterImpl)
			if !ok {
				continue
			}
			for _, s := range m.Samples {
				if sample, ok := s.(NumericMetricSample); ok {
					c.Inc(int64(sample.Value), sample.Labels)
				}
			}
		case "gauge":
			g, ok := r.Gauge(m.Name, m.Help).(*gaugeImpl)
			if !ok {
				continue
			}
			for _, s := range m.Samples {
				if sample, ok := s.(NumericMetricSample); ok {
					g.Set(sample.Value, sample.Labels)
				}
			}
		case "histogram":
			var buckets []float64
			if len(m.Samples) > 0 {
				if first, ok := m.Samples[0].(BucketMetricSample); ok {
					for _, b := range first.Buckets {
						buckets = append(buckets, bucketLeToFloat(b.Le))
					}
				}
			}
			h, ok := r.Histogram(m.Name, m.Help, buckets).(*histogramImpl)
			if !ok {
				continue
			}
			for _, s := range m.Samples {
				if sample, ok := s.(BucketMetricSample); ok {
					h.restore(sample)
				}
			}
		}
	}
}
