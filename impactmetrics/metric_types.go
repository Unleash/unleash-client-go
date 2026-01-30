package impactmetrics

import (
	"sort"
	"strings"
)

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
