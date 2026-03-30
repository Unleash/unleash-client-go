package unleash

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/stretchr/testify/assert"
)

type NoOpStorage struct{}

func (s *NoOpStorage) Persist(features *api.FeatureResponse) error {
	return nil
}

func (s *NoOpStorage) Load() (*api.FeatureResponse, error) {
	return &api.FeatureResponse{}, nil
}

func (s *NoOpStorage) Init(backupPath, appName string) {}

// testRepositoryListener is a test implementation of RepositoryListener that
// notifies via channels when events occur
type testRepositoryListener struct {
	updateChan chan bool
}

func (l *testRepositoryListener) OnReady() {
	// Not needed for this test
}

func (l *testRepositoryListener) OnUpdate() {
	if l.updateChan != nil {
		select {
		case l.updateChan <- true:
		default:
			// Channel is full, ignore
		}
	}
}

// mockSSEServer creates a test server that simulates SSE responses
// Since Gock doesn't support SSE streaming, we need a real test server for these tests
func mockSSEServer(hydrationData, updateData string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/client/register":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))

		case "/client/streaming":
			// Set SSE headers
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Send hydration event
			fmt.Fprintf(w, "event: unleash-connected\ndata: %s\n\n", hydrationData)
			w.(http.Flusher).Flush()

			// Send update event if provided
			if updateData != "" {
				time.Sleep(50 * time.Millisecond)
				fmt.Fprintf(w, "event: unleash-updated\ndata: %s\n\n", updateData)
				w.(http.Flusher).Flush()
			}

			// Keep connection open briefly to simulate streaming
			time.Sleep(100 * time.Millisecond)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// TestStreamingMode_UnleashConnectedEvent tests that the client can process an unleash-connected SSE event
func TestStreamingMode_UnleashConnectedEvent(t *testing.T) {
	hydrationData := `{"events":[{"type":"hydration","eventId":1,"features":[{"name":"test-feature","enabled":true,"strategies":[{"name":"default"}],"variants":[],"dependencies":null}],"segments":[]}]}`

	server := mockSSEServer(hydrationData, "")
	defer server.Close()

	// Create a listener for monitoring events
	listener := &testRepositoryListener{}

	// Configure client with streaming mode
	client, err := NewClient(
		WithUrl(server.URL),
		WithAppName("test-app"),
		WithInstanceId("test-instance"),
		WithCustomHeaders(http.Header{"X-API-KEY": []string{"123"}}),
		WithExperimentalMode(map[string]string{"type": "streaming"}),
		WithDisableMetrics(true),
		WithListener(listener),
	)
	assert.NoError(t, err)
	defer client.Close()

	// Wait for the client to connect and process the event
	client.WaitForReady()

	// Verify the feature is enabled after hydration
	assert.True(t, client.IsEnabled("test-feature", FeatureOptions{}), "test-feature should be enabled after hydration")

	// Test with a non-existent feature
	assert.False(t, client.IsEnabled("non-existent-feature", FeatureOptions{}), "non-existent-feature should be disabled")
}

// TestStreamingMode_UnleashUpdatedEvent tests processing of a simple unleash-updated event
func TestStreamingMode_UnleashUpdatedEvent(t *testing.T) {
	hydrationData := `{"events":[{"type":"hydration","eventId":1,"features":[{"name":"feature-1","enabled":false,"strategies":[{"name":"default"}]}],"segments":[]}]}`
	updateData := `{"events":[{"type":"feature-updated","eventId":2,"feature":{"name":"feature-1","enabled":true,"strategies":[{"name":"default"}]}}]}`

	server := mockSSEServer(hydrationData, updateData)
	defer server.Close()

	// Create a listener to detect updates
	updateChan := make(chan bool, 1)
	listener := &testRepositoryListener{
		updateChan: updateChan,
	}

	client, err := NewClient(
		WithUrl(server.URL),
		WithAppName("test-app"),
		WithInstanceId("test-instance"),
		WithDisableMetrics(true),
		WithExperimentalMode(map[string]string{"type": "streaming"}),
		WithStorage(&NoOpStorage{}),
		WithListener(listener),
	)
	assert.NoError(t, err)
	defer client.Close()

	// Wait for initial hydration
	client.WaitForReady()

	// Feature should initially be disabled
	assert.False(t, client.IsEnabled("feature-1", FeatureOptions{}), "feature-1 should be initially disabled")

	// Wait for the update event to be processed
	select {
	case <-updateChan:
		// Update received
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for update event")
	}

	// Feature should now be enabled after update
	assert.True(t, client.IsEnabled("feature-1", FeatureOptions{}), "feature-1 should be enabled after update")
}
