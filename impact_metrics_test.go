package unleash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5/internal/impactmetrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImpactMetricsSentInPayload(t *testing.T) {
	registry := impactmetrics.NewInMemoryMetricRegistry()
	labels := impactmetrics.MetricLabels{
		"appName":     "test-app",
		"environment": "test",
	}

	// Define and record all 3 metric types
	registry.Counter("purchases", "Number of purchases").Inc(1, labels)
	registry.Gauge("active_users", "Active users").Set(42, labels)
	registry.Histogram("latency", "Request latency", []float64{0.1, 0.5, 1.0}).Observe(0.3, labels)

	// Create and marshal payload like metrics service would
	payload := MetricsData{
		AppName:       "test-app",
		InstanceID:    "test-instance",
		ImpactMetrics: registry.Collect(),
	}

	jsonBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err)

	// Build metrics map by name
	metrics := make(map[string]map[string]interface{})
	for _, m := range result["impactMetrics"].([]interface{}) {
		metric := m.(map[string]interface{})
		metrics[metric["name"].(string)] = metric
	}

	expectedLabels := map[string]interface{}{
		"appName":     "test-app",
		"environment": "test",
	}

	assert.Equal(t, map[string]map[string]interface{}{
		"purchases": {
			"name": "purchases",
			"help": "Number of purchases",
			"type": "counter",
			"samples": []interface{}{
				map[string]interface{}{
					"labels": expectedLabels,
					"value":  float64(1),
				},
			},
		},
		"active_users": {
			"name": "active_users",
			"help": "Active users",
			"type": "gauge",
			"samples": []interface{}{
				map[string]interface{}{
					"labels": expectedLabels,
					"value":  float64(42),
				},
			},
		},
		"latency": {
			"name": "latency",
			"help": "Request latency",
			"type": "histogram",
			"samples": []interface{}{
				map[string]interface{}{
					"labels": expectedLabels,
					"count":  float64(1),
					"sum":    float64(0.3),
					"buckets": []interface{}{
						map[string]interface{}{"le": 0.1, "count": float64(0)},
						map[string]interface{}{"le": 0.5, "count": float64(1)},
						map[string]interface{}{"le": 1.0, "count": float64(1)},
						map[string]interface{}{"le": "+Inf", "count": float64(1)},
					},
				},
			},
		},
	}, metrics)
}

func TestImpactMetricsResentAfterFailure(t *testing.T) {
	type metricsPayload struct {
		Body []byte
		Path string
	}

	payloads := []metricsPayload{}
	payloadsMu := sync.Mutex{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/client/metrics" && r.Method == "POST" {
			// Capture request body
			body := make([]byte, r.ContentLength)
			r.Body.Read(body)
			r.Body.Close()

			payloadsMu.Lock()
			payloads = append(payloads, metricsPayload{Body: body, Path: r.URL.Path})
			isFirstRequest := len(payloads) == 1
			payloadsMu.Unlock()

			// First request fails with 500, second succeeds
			if isFirstRequest {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusAccepted)
			}
		} else if r.URL.Path == "/client/register" && r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithUrl(server.URL),
		WithAppName("test-app"),
		WithEnvironment("test"),
		WithMetricsInterval(50*time.Millisecond),
	)
	require.NoError(t, err)
	defer client.Close()

	api := client.ImpactMetrics()

	// Define and record metric
	api.DefineCounter("my_counter", "Test counter")
	api.IncrementCounterBy("my_counter", 5)

	// Wait for first send (should fail with 500)
	time.Sleep(100 * time.Millisecond)

	// Wait for second send (should succeed and include the restored metric)
	time.Sleep(100 * time.Millisecond)

	// Verify second request contains the metric
	payloadsMu.Lock()
	require.GreaterOrEqual(t, len(payloads), 2, "should have 2 metrics requests")

	// Parse second request body
	var payload map[string]interface{}
	err = json.Unmarshal(payloads[1].Body, &payload)
	payloadsMu.Unlock()

	require.NoError(t, err)

	// Verify metric is present in resent request
	impactMetrics := payload["impactMetrics"].([]interface{})
	require.Greater(t, len(impactMetrics), 0, "should have impact metrics in resent request")

	metric := impactMetrics[0].(map[string]interface{})
	assert.Equal(t, "my_counter", metric["name"])
}
