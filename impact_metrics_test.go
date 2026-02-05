package unleash

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImpactMetricsSentInPayload(t *testing.T) {
	payloads := make(chan []byte, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /client/metrics":
			body := make([]byte, r.ContentLength)
			r.Body.Read(body)
			r.Body.Close()

			payloads <- body

			w.WriteHeader(http.StatusAccepted)
		case "POST /client/register":
			w.WriteHeader(http.StatusOK)
		default:
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

	// Define and record all 3 metric types using public API
	api.DefineCounter("purchases", "Number of purchases")
	api.IncrementCounter("purchases")

	api.DefineGauge("active_users", "Active users")
	api.UpdateGauge("active_users", 42)

	api.DefineHistogram("latency", "Request latency", 0.1, 0.5, 1.0)
	api.ObserveHistogram("latency", 0.3)

	// Wait for metrics to be collected and sent
	var payload map[string]interface{}
	err = json.Unmarshal(<-payloads, &payload)

	require.NoError(t, err)

	// Build metrics map by name
	metrics := make(map[string]map[string]interface{})
	impactMetrics := payload["impactMetrics"].([]interface{})
	for _, m := range impactMetrics {
		metric := m.(map[string]interface{})
		metrics[metric["name"].(string)] = metric
	}

	expectedLabels := map[string]interface{}{
		"appName":     "test-app",
		"environment": "test",
	}

	// Verify complete payload structure
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
	payloads := make(chan []byte, 2)
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "POST /client/metrics":
			body := make([]byte, r.ContentLength)
			r.Body.Read(body)
			r.Body.Close()

			payloads <- body

			requestCount.Add(1)
			isFirstRequest := requestCount.Load() == 1

			// First request fails with 500, second succeeds
			if isFirstRequest {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusAccepted)
			}
		case "POST /client/register":
			w.WriteHeader(http.StatusOK)
		default:
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

	// Wait for both requests: first fails, second succeeds
	<-payloads // First request (fails)
	secondPayload := <-payloads // Second request (succeeds)

	// Verify second request contains the metric
	var payload map[string]interface{}
	err = json.Unmarshal(secondPayload, &payload)

	require.NoError(t, err)

	// Verify metric is present in resent request
	impactMetrics := payload["impactMetrics"].([]interface{})
	require.Greater(t, len(impactMetrics), 0, "should have impact metrics in resent request")

	metric := impactMetrics[0].(map[string]interface{})
	assert.Equal(t, "my_counter", metric["name"])
}
