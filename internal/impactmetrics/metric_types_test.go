package impactmetrics

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCounterIncrementsByDefaultValue(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	counter := registry.Counter("test_counter", "testing")

	counter.Inc(1, nil)

	result := registry.Collect()
	metric := findMetric(result, "test_counter")

	assert.Equal(t, CollectedMetric{
		Name: "test_counter",
		Help: "testing",
		Type: "counter",
		Samples: []interface{}{
			CounterMetricSample{Labels: MetricLabels{}, Value: 1},
		},
	}, *metric)
}

func TestCounterIncrementsWithCustomValueAndLabels(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	counter := registry.Counter("labeled_counter", "with labels")

	counter.Inc(3, MetricLabels{"foo": "bar"})
	counter.Inc(-1, MetricLabels{"foo": "bar"})
	counter.Inc(0, MetricLabels{"foo": "bar"})
	counter.Inc(2, MetricLabels{"foo": "bar"})

	result := registry.Collect()
	metric := findMetric(result, "labeled_counter")

	assert.Equal(t, CollectedMetric{
		Name: "labeled_counter",
		Help: "with labels",
		Type: "counter",
		Samples: []interface{}{
			CounterMetricSample{Labels: MetricLabels{"foo": "bar"}, Value: 5},
		},
	}, *metric)
}

func TestDifferentLabelCombinationsAreStoredSeparately(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	counter := registry.Counter("multi_label", "label test")

	counter.Inc(1, MetricLabels{"a": "x"})
	counter.Inc(2, MetricLabels{"b": "y"})
	counter.Inc(3, nil)

	result := registry.Collect()
	metric := findMetric(result, "multi_label")

	assert.Equal(t, CollectedMetric{
		Name: "multi_label",
		Help: "label test",
		Type: "counter",
		Samples: []interface{}{
			CounterMetricSample{Labels: MetricLabels{}, Value: 3},
			CounterMetricSample{Labels: MetricLabels{"a": "x"}, Value: 1},
			CounterMetricSample{Labels: MetricLabels{"b": "y"}, Value: 2},
		},
	}, *metric)
}

func TestGaugeSupportsIncDecAndSet(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	gauge := registry.Gauge("test_gauge", "gauge test")

	gauge.Inc(5, MetricLabels{"env": "prod"})
	gauge.Dec(2, MetricLabels{"env": "prod"})
	gauge.Set(10, MetricLabels{"env": "prod"})

	result := registry.Collect()
	metric := findMetric(result, "test_gauge")

	assert.Equal(t, CollectedMetric{
		Name: "test_gauge",
		Help: "gauge test",
		Type: "gauge",
		Samples: []interface{}{
			GaugeMetricSample{Labels: MetricLabels{"env": "prod"}, Value: 10},
		},
	}, *metric)
}

func TestGaugeTracksValuesSeparatelyPerLabelSet(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	gauge := registry.Gauge("multi_env_gauge", "tracks multiple envs")

	gauge.Inc(5, MetricLabels{"env": "prod"})
	gauge.Dec(2, MetricLabels{"env": "dev"})
	gauge.Set(10, MetricLabels{"env": "test"})

	result := registry.Collect()
	metric := findMetric(result, "multi_env_gauge")

	assert.Equal(t, CollectedMetric{
		Name: "multi_env_gauge",
		Help: "tracks multiple envs",
		Type: "gauge",
		Samples: []interface{}{
			GaugeMetricSample{Labels: MetricLabels{"env": "dev"}, Value: -2},
			GaugeMetricSample{Labels: MetricLabels{"env": "prod"}, Value: 5},
			GaugeMetricSample{Labels: MetricLabels{"env": "test"}, Value: 10},
		},
	}, *metric)
}

func TestCollectReturnsCounterWithZeroValueWhenCounterIsEmpty(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	registry.Counter("noop_counter", "noop")
	registry.Gauge("noop_gauge", "noop")

	result := registry.Collect()

	assert.Equal(t, []CollectedMetric{
		{
			Name: "noop_counter",
			Help: "noop",
			Type: "counter",
			Samples: []interface{}{
				CounterMetricSample{Labels: MetricLabels{}, Value: 0},
			},
		},
	}, result)
}

func TestCollectReturnsCounterWithZeroValueAfterFlushingPreviousValues(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	counter := registry.Counter("flush_test", "flush")

	counter.Inc(1, nil)
	first := registry.Collect()
	assert.Len(t, first, 1)

	second := registry.Collect()
	assert.Equal(t, []CollectedMetric{
		{
			Name: "flush_test",
			Help: "flush",
			Type: "counter",
			Samples: []interface{}{
				CounterMetricSample{Labels: MetricLabels{}, Value: 0},
			},
		},
	}, second)
}

func TestRestoreReinsertsCollectedMetricsIntoTheRegistry(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	counter := registry.Counter("restore_test", "testing restore")

	counter.Inc(5, MetricLabels{"tag": "a"})
	counter.Inc(2, MetricLabels{"tag": "b"})

	flushed := registry.Collect()
	assert.Len(t, flushed, 1)

	afterFlush := registry.Collect()
	assert.Equal(t, []CollectedMetric{
		{
			Name: "restore_test",
			Help: "testing restore",
			Type: "counter",
			Samples: []interface{}{
				CounterMetricSample{Labels: MetricLabels{}, Value: 0},
			},
		},
	}, afterFlush)

	registry.Restore(flushed)

	restored := registry.Collect()
	assert.Equal(t, []CollectedMetric{
		{
			Name: "restore_test",
			Help: "testing restore",
			Type: "counter",
			Samples: []interface{}{
				CounterMetricSample{Labels: MetricLabels{"tag": "a"}, Value: 5},
				CounterMetricSample{Labels: MetricLabels{"tag": "b"}, Value: 2},
			},
		},
	}, restored)
}

func TestHistogramObservesValues(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	histogram := registry.Histogram("test_histogram", "testing histogram", []float64{0.1, 0.5, 1, 2.5, 5})

	histogram.Observe(0.05, MetricLabels{"env": "prod"})
	histogram.Observe(0.75, MetricLabels{"env": "prod"})
	histogram.Observe(3, MetricLabels{"env": "prod"})

	result := registry.Collect()

	assert.Equal(t, []CollectedMetric{
		{
			Name: "test_histogram",
			Help: "testing histogram",
			Type: "histogram",
			Samples: []interface{}{
				HistogramMetricSample{
					Labels: MetricLabels{"env": "prod"},
					Count:  3,
					Sum:    3.8,
					Buckets: []BucketEntry{
						{Le: 0.1, Count: 1},
						{Le: 0.5, Count: 1},
						{Le: 1.0, Count: 2},
						{Le: 2.5, Count: 2},
						{Le: 5.0, Count: 3},
						{Le: "+Inf", Count: 3},
					},
				},
			},
		},
	}, result)
}

func TestHistogramTracksDifferentLabelCombinationsSeparately(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	histogram := registry.Histogram("multi_label_histogram", "histogram with multiple labels", []float64{1, 10})

	histogram.Observe(0.5, MetricLabels{"method": "GET"})
	histogram.Observe(5, MetricLabels{"method": "POST"})
	histogram.Observe(15, nil)

	result := registry.Collect()

	assert.Equal(t, []CollectedMetric{
		{
			Name: "multi_label_histogram",
			Help: "histogram with multiple labels",
			Type: "histogram",
			Samples: []interface{}{
				HistogramMetricSample{
					Labels: MetricLabels{},
					Count:  1,
					Sum:    15.0,
					Buckets: []BucketEntry{
						{Le: 1.0, Count: 0},
						{Le: 10.0, Count: 0},
						{Le: "+Inf", Count: 1},
					},
				},
				HistogramMetricSample{
					Labels: MetricLabels{"method": "GET"},
					Count:  1,
					Sum:    0.5,
					Buckets: []BucketEntry{
						{Le: 1.0, Count: 1},
						{Le: 10.0, Count: 1},
						{Le: "+Inf", Count: 1},
					},
				},
				HistogramMetricSample{
					Labels: MetricLabels{"method": "POST"},
					Count:  1,
					Sum:    5.0,
					Buckets: []BucketEntry{
						{Le: 1.0, Count: 0},
						{Le: 10.0, Count: 1},
						{Le: "+Inf", Count: 1},
					},
				},
			},
		},
	}, result)
}

func TestHistogramRestorationPreservesExactData(t *testing.T) {
	registry := NewInMemoryMetricRegistry()
	histogram := registry.Histogram("restore_histogram", "testing histogram restore", []float64{0.1, 1, 10})

	histogram.Observe(0.05, MetricLabels{"method": "GET"})
	histogram.Observe(0.5, MetricLabels{"method": "GET"})
	histogram.Observe(5, MetricLabels{"method": "POST"})
	histogram.Observe(15, MetricLabels{"method": "POST"})

	firstCollect := registry.Collect()
	assert.Len(t, firstCollect, 1)

	emptyCollect := registry.Collect()
	assert.Equal(t, []CollectedMetric{
		{
			Name: "restore_histogram",
			Help: "testing histogram restore",
			Type: "histogram",
			Samples: []interface{}{
				HistogramMetricSample{
					Labels: MetricLabels{},
					Count:  0,
					Sum:    0,
					Buckets: []BucketEntry{
						{Le: 0.1, Count: 0},
						{Le: 1.0, Count: 0},
						{Le: 10.0, Count: 0},
						{Le: "+Inf", Count: 0},
					},
				},
			},
		},
	}, emptyCollect)

	registry.Restore(firstCollect)

	restoredCollect := registry.Collect()
	assert.Equal(t, firstCollect, restoredCollect)
}

func TestAllMetricOperationsSilentlyDropInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		value float64
	}{
		{"Infinity", math.Inf(1)},
		{"-Infinity", math.Inf(-1)},
		{"NaN", math.NaN()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewInMemoryMetricRegistry()
			counter := registry.Counter("c", "h")
			gauge := registry.Gauge("g", "h")
			histogram := registry.Histogram("h", "h", []float64{1})

			counter.Inc(1, nil)
			gauge.Set(5, nil)
			gauge.Set(tt.value, nil)
			gauge.Inc(tt.value, nil)
			gauge.Dec(tt.value, nil)
			histogram.Observe(0.5, nil)
			histogram.Observe(tt.value, nil)

			result := registry.Collect()

			assert.Equal(t, []CollectedMetric{
				{Name: "c", Help: "h", Type: "counter", Samples: []interface{}{
					CounterMetricSample{Labels: MetricLabels{}, Value: 1},
				}},
				{Name: "g", Help: "h", Type: "gauge", Samples: []interface{}{
					GaugeMetricSample{Labels: MetricLabels{}, Value: 5},
				}},
				{Name: "h", Help: "h", Type: "histogram", Samples: []interface{}{
					HistogramMetricSample{
						Labels: MetricLabels{},
						Count:  1,
						Sum:    0.5,
						Buckets: []BucketEntry{
							{Le: 1.0, Count: 1},
							{Le: "+Inf", Count: 1},
						},
					},
				}},
			}, result)
		})
	}
}

func findMetric(metrics []CollectedMetric, name string) *CollectedMetric {
	for _, m := range metrics {
		if m.Name == name {
			return &m
		}
	}
	return nil
}
