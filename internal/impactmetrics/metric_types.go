package impactmetrics

import (
	"encoding/json"
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

type BucketEntry struct {
	Le    float64
	Count int64
}

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
	}{
		Le:    leValue,
		Count: be.Count,
	})
}

type HistogramMetricSample struct {
	Labels  MetricLabels  `json:"labels"`
	Count   int64         `json:"count"`
	Sum     float64       `json:"sum"`
	Buckets []BucketEntry `json:"buckets"`
}

type Sample interface {
	isSample()
}

func (CounterMetricSample) isSample()   {}
func (GaugeMetricSample) isSample()     {}
func (HistogramMetricSample) isSample() {}

type CollectedMetric struct {
	Name    string   `json:"name"`
	Help    string   `json:"help"`
	Type    string   `json:"type"`
	Samples []Sample `json:"samples"`
}

type CollectedMetrics []CollectedMetric

func (cm CollectedMetrics) IsEmpty() bool {
	return len(cm) == 0
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
	key := LabelKey(labels)
	c.mu.Lock()
	defer c.mu.Unlock()
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

	samples := make([]Sample, 0, len(c.values))
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
	key := LabelKey(labels)
	g.mu.Lock()
	defer g.mu.Unlock()
	g.values[key] += value
}

func (g *gaugeImpl) Dec(value float64, labels MetricLabels) {
	if isInvalidValue(value) {
		return
	}
	key := LabelKey(labels)
	g.mu.Lock()
	defer g.mu.Unlock()
	g.values[key] -= value
}

func (g *gaugeImpl) Set(value float64, labels MetricLabels) {
	if isInvalidValue(value) {
		return
	}
	key := LabelKey(labels)
	g.mu.Lock()
	defer g.mu.Unlock()
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

	samples := make([]Sample, 0, len(g.values))
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

type Histogram interface {
	Observe(value float64, labels MetricLabels)
	Restore(sample HistogramMetricSample)
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
}

func newHistogram(name, help string, buckets []float64) *histogramImpl {
	if len(buckets) == 0 {
		buckets = append([]float64(nil), DefaultHistogramBuckets...)
	}
	sortedBuckets := uniqueSorted(buckets)
	sortedBuckets = append(sortedBuckets, math.Inf(1))

	return &histogramImpl{
		name:    name,
		help:    help,
		buckets: sortedBuckets,
		values:  map[string]*histogramData{},
	}
}

func uniqueSorted(buckets []float64) []float64 {
	var filtered []float64
	for _, b := range buckets {
		if b != math.Inf(1) {
			filtered = append(filtered, b)
		}
	}
	sort.Float64s(filtered)

	// Deduplicate adjacent elements
	if len(filtered) == 0 {
		return filtered
	}
	result := filtered[:1]
	for i := 1; i < len(filtered); i++ {
		if filtered[i] != filtered[i-1] {
			result = append(result, filtered[i])
		}
	}
	return result
}

func (h *histogramImpl) Observe(value float64, labels MetricLabels) {
	if isInvalidValue(value) {
		return
	}

	key := LabelKey(labels)
	h.mu.Lock()
	defer h.mu.Unlock()
	data, ok := h.values[key]
	if !ok {
		data = &histogramData{
			buckets: make(map[float64]int64),
		}
		h.values[key] = data
	}

	data.count++
	data.sum += value
	for _, b := range h.buckets {
		if value <= b {
			data.buckets[b]++
		}
	}
}

func (h *histogramImpl) Restore(sample HistogramMetricSample) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := LabelKey(sample.Labels)
	data := &histogramData{
		count:   sample.Count,
		sum:     sample.Sum,
		buckets: make(map[float64]int64, len(sample.Buckets)),
	}
	for _, b := range sample.Buckets {
		data.buckets[b.Le] = b.Count
	}
	h.values[key] = data
}

func (h *histogramImpl) defaultHistogramSample() HistogramMetricSample {
	bucketEntries := make([]BucketEntry, len(h.buckets))
	for i, b := range h.buckets {
		bucketEntries[i] = BucketEntry{
			Le:    b,
			Count: 0,
		}
	}
	return HistogramMetricSample{
		Labels:  MetricLabels{},
		Count:   0,
		Sum:     0,
		Buckets: bucketEntries,
	}
}

func (h *histogramImpl) collect() CollectedMetric {
	h.mu.Lock()
	defer h.mu.Unlock()

	keys := make([]string, 0, len(h.values))
	for k := range h.values {
		keys = append(keys, k)
	}
	// Sort keys for deterministic output since Go map iteration order is random
	sort.Strings(keys)

	samples := make([]Sample, 0, len(h.values))
	for _, key := range keys {
		data := h.values[key]
		bucketEntries := make([]BucketEntry, len(h.buckets))
		for i, b := range h.buckets {
			bucketEntries[i] = BucketEntry{
				Le:    b,
				Count: data.buckets[b],
			}
		}
		samples = append(samples, HistogramMetricSample{
			Labels:  ParseLabelKey(key),
			Count:   data.count,
			Sum:     data.sum,
			Buckets: bucketEntries,
		})
	}

	h.values = map[string]*histogramData{}

	if len(samples) == 0 {
		samples = append(samples, h.defaultHistogramSample())
	}

	return CollectedMetric{
		Name:    h.name,
		Help:    h.help,
		Type:    "histogram",
		Samples: samples,
	}
}

type InMemoryMetricRegistry struct {
	mu         sync.RWMutex
	counters   map[string]*counterImpl
	gauges     map[string]*gaugeImpl
	histograms map[string]*histogramImpl
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
	if c, ok := r.counters[name]; ok {
		return c
	}
	c := newCounter(name, help)
	r.counters[name] = c
	return c
}

func (r *InMemoryMetricRegistry) GetCounter(name string) Counter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c, ok := r.counters[name]; ok {
		return c
	}
	return nil
}

func (r *InMemoryMetricRegistry) Gauge(name, help string) Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	if g, ok := r.gauges[name]; ok {
		return g
	}
	g := newGauge(name, help)
	r.gauges[name] = g
	return g
}

func (r *InMemoryMetricRegistry) GetGauge(name string) Gauge {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if g, ok := r.gauges[name]; ok {
		return g
	}
	return nil
}

func (r *InMemoryMetricRegistry) Histogram(name, help string, buckets []float64) Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.histograms[name]; ok {
		return h
	}
	h := newHistogram(name, help, buckets)
	r.histograms[name] = h
	return h
}

func (r *InMemoryMetricRegistry) GetHistogram(name string) Histogram {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if h, ok := r.histograms[name]; ok {
		return h
	}
	return nil
}

func (r *InMemoryMetricRegistry) Collect() []CollectedMetric {
	r.mu.Lock()
	defer r.mu.Unlock()

	counterNames := make([]string, 0, len(r.counters))
	for name := range r.counters {
		counterNames = append(counterNames, name)
	}
	// Sort names for deterministic output since Go map iteration order is random
	sort.Strings(counterNames)

	var result []CollectedMetric
	for _, name := range counterNames {
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

	histogramNames := make([]string, 0, len(r.histograms))
	for name := range r.histograms {
		histogramNames = append(histogramNames, name)
	}
	// Sort names for deterministic output since Go map iteration order is random
	sort.Strings(histogramNames)

	for _, name := range histogramNames {
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
		case "histogram":
			var firstSampleBuckets []float64
			for _, s := range m.Samples {
				if sample, ok := s.(HistogramMetricSample); ok {
					for _, b := range sample.Buckets {
						firstSampleBuckets = append(firstSampleBuckets, b.Le)
					}
					break
				}
			}

			h := r.Histogram(m.Name, m.Help, firstSampleBuckets)

			for _, s := range m.Samples {
				if sample, ok := s.(HistogramMetricSample); ok {
					h.Restore(sample)
				}
			}
		}
	}
}
