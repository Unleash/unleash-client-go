package unleash

import (
	"encoding/json"
	"io"
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
		if r.Method == "POST" && r.URL.Path == "/client/metrics" {
			body, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err == nil {
				payloads <- body
			}
			w.WriteHeader(http.StatusAccepted)
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

	api.DefineCounter("purchases", "Number of purchases")
	api.IncrementCounter("purchases")

	api.DefineGauge("active_users", "Active users")
	api.UpdateGauge("active_users", 42)

	api.DefineHistogram("latency", "Request latency", 0.1, 0.5, 1.0)
	api.ObserveHistogram("latency", 0.3)

	payloadBytes := <-payloads
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(payloadBytes, &payload))

	impactMetrics := payload["impactMetrics"].([]interface{})
	impactMetricsJSON, err := json.Marshal(impactMetrics)
	require.NoError(t, err)

	expectedJSON := `[
		{
			"name": "purchases",
			"help": "Number of purchases",
			"type": "counter",
			"samples": [
				{
					"labels": {"appName": "test-app", "environment": "test"},
					"value": 1
				}
			]
		},
		{
			"name": "active_users",
			"help": "Active users",
			"type": "gauge",
			"samples": [
				{
					"labels": {"appName": "test-app", "environment": "test"},
					"value": 42
				}
			]
		},
		{
			"name": "latency",
			"help": "Request latency",
			"type": "histogram",
			"samples": [
				{
					"labels": {"appName": "test-app", "environment": "test"},
					"count": 1,
					"sum": 0.3,
					"buckets": [
						{"le": 0.1, "count": 0},
						{"le": 0.5, "count": 1},
						{"le": 1.0, "count": 1},
						{"le": "+Inf", "count": 1}
					]
				}
			]
		}
	]`

	assert.JSONEq(t, expectedJSON, string(impactMetricsJSON))
}

func TestImpactMetricsResentAfterFailure(t *testing.T) {
	payloads := make(chan []byte, 2)
	var failNext atomic.Bool
	failNext.Store(true)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/client/metrics" {
			body, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err == nil {
				payloads <- body
			}

			if failNext.Load() {
				failNext.Store(false)
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusAccepted)
			}
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

	// Wait for both requests: first fails, second succeeds
	<-payloads                  // First request (fails)
	secondPayload := <-payloads // Second request (succeeds)

	// Verify second request contains the metric
	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(secondPayload, &payload))

	impactMetrics := payload["impactMetrics"].([]interface{})
	require.Greater(t, len(impactMetrics), 0, "should have impact metrics in resent request")

	metric := impactMetrics[0].(map[string]interface{})
	assert.Equal(t, "my_counter", metric["name"])
}
