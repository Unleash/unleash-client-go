package unleash

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestStreamingMode_UnleashConnectedEvent tests that the client can process an unleash-connected SSE event
func TestStreamingMode_UnleashConnectedEvent(t *testing.T) {
	// Track requests to verify the client connects to the streaming endpoint
	var streamingRequests int32
	var registerRequests int32

	// Create a test server that mocks the Unleash server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/client/register":
			atomic.AddInt32(&registerRequests, 1)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))

		case "/client/streaming":
			atomic.AddInt32(&streamingRequests, 1)
			
			// Verify headers
			assert.Equal(t, "test-app", r.Header.Get("UNLEASH-APPNAME"))
			assert.Equal(t, "test-instance", r.Header.Get("UNLEASH-INSTANCEID"))
			assert.Equal(t, "123", r.Header.Get("X-API-KEY"))

			// Set SSE headers
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			
			// Send SSE response with unleash-connected event containing hydration
			sseData := `event: unleash-connected
data: {"events":[{"type":"hydration","eventId":1,"features":[{"name":"test-feature","enabled":true,"strategies":[{"name":"default"}],"variants":[],"dependencies":null}],"segments":[]}]}

`
			w.Write([]byte(sseData))
			
			// Keep connection open for a bit to simulate streaming
			time.Sleep(100 * time.Millisecond)

		default:
			t.Errorf("Unexpected request to %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Configure client with streaming mode
	client, err := NewClient(
		WithUrl(server.URL),
		WithAppName("test-app"),
		WithInstanceId("test-instance"),
		WithCustomHeaders(http.Header{"X-API-KEY": []string{"123"}}),
		WithExperimentalMode(map[string]string{"type": "streaming"}),
		WithStorage(NewDefaultDeltaStorage()), // Use DeltaStorage for streaming
	)
	assert.NoError(t, err)
	defer client.Close()

	// Wait for the client to connect and process the event
	client.WaitForReady()

	// Verify the streaming endpoint was called
	assert.Greater(t, atomic.LoadInt32(&streamingRequests), int32(0), "Streaming endpoint should have been called")

	// Verify the feature is enabled after hydration
	isEnabled := client.IsEnabled("test-feature")
	assert.True(t, isEnabled, "test-feature should be enabled after hydration")

	// Test with a non-existent feature
	isDisabled := client.IsEnabled("non-existent-feature")
	assert.False(t, isDisabled, "non-existent-feature should be disabled")
}

// TestStreamingMode_UnleashUpdatedEvent tests processing of unleash-updated events
func TestStreamingMode_UnleashUpdatedEvent(t *testing.T) {
	var streamingRequests int32
	eventsSent := make(chan string, 10)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/client/register":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))

		case "/client/streaming":
			atomic.AddInt32(&streamingRequests, 1)
			
			// Set SSE headers
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")
			
			// First send hydration to initialize
			hydrationData := `event: unleash-connected
data: {"events":[{"type":"hydration","eventId":1,"features":[{"name":"feature-1","enabled":false,"strategies":[{"name":"default"}]}],"segments":[]}]}

`
			w.Write([]byte(hydrationData))
			w.(http.Flusher).Flush()
			eventsSent <- "hydration"
			
			// Wait a bit then send an update
			time.Sleep(50 * time.Millisecond)
			
			// Send update event to enable the feature
			updateData := `event: unleash-updated
data: {"events":[{"type":"feature-updated","eventId":2,"feature":{"name":"feature-1","enabled":true,"strategies":[{"name":"default"}]}}]}

`
			w.Write([]byte(updateData))
			w.(http.Flusher).Flush()
			eventsSent <- "update"
			
			// Keep connection open
			time.Sleep(100 * time.Millisecond)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithUrl(server.URL),
		WithAppName("test-app"),
		WithInstanceId("test-instance"),
		WithDisableMetrics(true),
		WithExperimentalMode(map[string]string{"type": "streaming"}),
		WithStorage(NewDefaultDeltaStorage()),
		WithRefreshInterval(time.Hour), // Long interval to ensure we're using streaming
	)
	assert.NoError(t, err)
	defer client.Close()

	// Wait for initial hydration
	client.WaitForReady()
	
	// Feature should initially be disabled
	assert.False(t, client.IsEnabled("feature-1"), "feature-1 should be initially disabled")

	// Wait for the update event to be processed
	select {
	case event := <-eventsSent:
		assert.Equal(t, "hydration", event)
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for hydration event")
	}

	select {
	case event := <-eventsSent:
		assert.Equal(t, "update", event)
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for update event")
	}

	// Give a bit more time for processing
	time.Sleep(100 * time.Millisecond)

	// Feature should now be enabled after update
	assert.True(t, client.IsEnabled("feature-1"), "feature-1 should be enabled after update")
	
	// Verify streaming was used
	assert.Greater(t, atomic.LoadInt32(&streamingRequests), int32(0), "Streaming endpoint should have been called")
}

// TestStreamingMode_MultipleDeltaEvents tests processing multiple delta events in sequence
func TestStreamingMode_MultipleDeltaEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/client/register":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))

		case "/client/streaming":
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			
			// Initial hydration with two features
			hydration := `event: unleash-connected
data: {"events":[{"type":"hydration","eventId":1,"features":[{"name":"feature-a","enabled":true,"strategies":[{"name":"default"}]},{"name":"feature-b","enabled":true,"strategies":[{"name":"default"}]}],"segments":[]}]}

`
			w.Write([]byte(hydration))
			w.(http.Flusher).Flush()
			
			time.Sleep(50 * time.Millisecond)
			
			// Update feature-a to disabled
			update1 := `event: unleash-updated
data: {"events":[{"type":"feature-updated","eventId":2,"feature":{"name":"feature-a","enabled":false,"strategies":[{"name":"default"}]}}]}

`
			w.Write([]byte(update1))
			w.(http.Flusher).Flush()
			
			time.Sleep(50 * time.Millisecond)
			
			// Add new feature-c
			update2 := `event: unleash-updated
data: {"events":[{"type":"feature-updated","eventId":3,"feature":{"name":"feature-c","enabled":true,"strategies":[{"name":"default"}]}}]}

`
			w.Write([]byte(update2))
			w.(http.Flusher).Flush()
			
			time.Sleep(50 * time.Millisecond)
			
			// Remove feature-b
			update3 := `event: unleash-updated
data: {"events":[{"type":"feature-removed","eventId":4,"featureName":"feature-b","project":"default"}]}

`
			w.Write([]byte(update3))
			w.(http.Flusher).Flush()
			
			// Keep connection open
			time.Sleep(100 * time.Millisecond)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithUrl(server.URL),
		WithAppName("test-app"),
		WithInstanceId("test-instance"),
		WithDisableMetrics(true),
		WithExperimentalMode(map[string]string{"type": "streaming"}),
		WithStorage(NewDefaultDeltaStorage()),
	)
	assert.NoError(t, err)
	defer client.Close()

	// Wait for initial hydration
	client.WaitForReady()
	
	// Initial state after hydration
	assert.True(t, client.IsEnabled("feature-a"), "feature-a should be initially enabled")
	assert.True(t, client.IsEnabled("feature-b"), "feature-b should be initially enabled")
	assert.False(t, client.IsEnabled("feature-c"), "feature-c should not exist initially")

	// Wait for all updates to be processed
	time.Sleep(300 * time.Millisecond)

	// Final state after all updates
	assert.False(t, client.IsEnabled("feature-a"), "feature-a should be disabled after update")
	assert.False(t, client.IsEnabled("feature-b"), "feature-b should be removed")
	assert.True(t, client.IsEnabled("feature-c"), "feature-c should be added and enabled")
}

// TestStreamingMode_SegmentUpdates tests segment updates via streaming
func TestStreamingMode_SegmentUpdates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/client/register":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))

		case "/client/streaming":
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			
			// Hydration with feature using segments
			hydration := `event: unleash-connected
data: {"events":[{"type":"hydration","eventId":1,"features":[{"name":"segmented-feature","enabled":true,"strategies":[{"name":"default","segments":[1]}]}],"segments":[{"id":1,"constraints":[{"contextName":"userId","operator":"IN","values":["123"]}]}]}]}

`
			w.Write([]byte(hydration))
			w.(http.Flusher).Flush()
			
			// Give more time for client to process hydration
			time.Sleep(200 * time.Millisecond)
			
			// Update segment to include more users
			segmentUpdate := `event: unleash-updated
data: {"events":[{"type":"segment-updated","eventId":2,"segment":{"id":1,"constraints":[{"contextName":"userId","operator":"IN","values":["123","456"]}]}}]}

`
			w.Write([]byte(segmentUpdate))
			w.(http.Flusher).Flush()
			
			time.Sleep(200 * time.Millisecond)
			
			// Add a new segment and update feature to use it
			newSegment := `event: unleash-updated
data: {"events":[{"type":"segment-updated","eventId":3,"segment":{"id":2,"constraints":[{"contextName":"environment","operator":"IN","values":["production"]}]}},{"type":"feature-updated","eventId":4,"feature":{"name":"segmented-feature","enabled":true,"strategies":[{"name":"default","segments":[1,2]}]}}]}

`
			w.Write([]byte(newSegment))
			w.(http.Flusher).Flush()
			
			// Keep connection open longer to ensure all events are received
			time.Sleep(1000 * time.Millisecond)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(
		WithUrl(server.URL),
		WithAppName("test-app"),
		WithInstanceId("test-instance"),
		WithDisableMetrics(true),
		WithExperimentalMode(map[string]string{"type": "streaming"}),
		WithStorage(NewDefaultDeltaStorage()),
	)
	assert.NoError(t, err)
	defer client.Close()

	// Wait for initial hydration
	client.WaitForReady()

	// Test with user 123 (should be enabled due to segment)
	ctx := &context.Context{UserId: "123"}
	assert.True(t, client.IsEnabled("segmented-feature", WithContext(*ctx)), 
		"segmented-feature should be enabled for user 123")

	// Test with user 456 (initially not in segment)
	ctx456 := &context.Context{UserId: "456"}
	assert.False(t, client.IsEnabled("segmented-feature", WithContext(*ctx456)), 
		"segmented-feature should initially be disabled for user 456")

	// Wait for first segment update to be processed
	time.Sleep(400 * time.Millisecond)

	// After first segment update, user 456 should now be included (still only requires segment 1)
	assert.True(t, client.IsEnabled("segmented-feature", WithContext(*ctx456)), 
		"segmented-feature should be enabled for user 456 after first segment update")
	
	// Wait for second update (new segment + feature update)
	time.Sleep(400 * time.Millisecond)
	
	// After feature is updated to require both segments, user 456 alone is NOT enough
	assert.False(t, client.IsEnabled("segmented-feature", WithContext(*ctx456)), 
		"segmented-feature should be disabled for user 456 after feature requires both segments")

	// Test with both constraints after new segment is added
	// Note: When a feature has multiple segments, ALL segment constraints must be satisfied
	ctxProd := &context.Context{
		UserId:      "123",         // Satisfies segment 1
		Environment: "production",  // Satisfies segment 2 (use Environment field, not Properties)
	}
	assert.True(t, client.IsEnabled("segmented-feature", WithContext(*ctxProd)),
		"segmented-feature should be enabled with both segment constraints satisfied")

	// Test with user not in segment
	ctxOther := &context.Context{UserId: "999"}
	assert.False(t, client.IsEnabled("segmented-feature", WithContext(*ctxOther)),
		"segmented-feature should be disabled for user not in segment")
}

// TestStreamingMode_ConnectionHeaders tests that the correct headers are sent
func TestStreamingMode_ConnectionHeaders(t *testing.T) {
	headerChecks := make(map[string]string)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/client/register":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))

		case "/client/streaming":
			// Capture headers for verification
			headerChecks["UNLEASH-APPNAME"] = r.Header.Get("UNLEASH-APPNAME")
			headerChecks["UNLEASH-INSTANCEID"] = r.Header.Get("UNLEASH-INSTANCEID")
			headerChecks["X-CUSTOM-HEADER"] = r.Header.Get("X-CUSTOM-HEADER")
			
			w.Header().Set("Content-Type", "text/event-stream")
			
			// Send minimal hydration
			hydration := `event: unleash-connected
data: {"events":[{"type":"hydration","eventId":1,"features":[],"segments":[]}]}

`
			w.Write([]byte(hydration))
			time.Sleep(50 * time.Millisecond)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	customHeaders := http.Header{
		"X-CUSTOM-HEADER": []string{"custom-value"},
	}

	client, err := NewClient(
		WithUrl(server.URL),
		WithAppName("my-app"),
		WithInstanceId("my-instance"),
		WithDisableMetrics(true),
		WithCustomHeaders(customHeaders),
		WithExperimentalMode(map[string]string{"type": "streaming"}),
		WithStorage(NewDefaultDeltaStorage()),
	)
	assert.NoError(t, err)
	defer client.Close()

	client.WaitForReady()

	// Verify headers were sent correctly
	assert.Equal(t, "my-app", headerChecks["UNLEASH-APPNAME"])
	assert.Equal(t, "my-instance", headerChecks["UNLEASH-INSTANCEID"])
	assert.Equal(t, "custom-value", headerChecks["X-CUSTOM-HEADER"])
}

// TestStreamingMode_ErrorHandling tests error handling in streaming mode
func TestStreamingMode_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/client/register":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))

		case "/client/streaming":
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			
			flusher := w.(http.Flusher)
			
			// Send valid hydration first
			hydration := `event: unleash-connected
data: {"events":[{"type":"hydration","eventId":1,"features":[{"name":"test","enabled":true,"strategies":[]}],"segments":[]}]}

`
			w.Write([]byte(hydration))
			flusher.Flush()
			
			time.Sleep(100 * time.Millisecond)
			
			// Send malformed event (should be ignored)
			malformed := `event: unleash-updated
data: {invalid json}

`
			w.Write([]byte(malformed))
			flusher.Flush()
			
			time.Sleep(100 * time.Millisecond)
			
			// Send valid update
			update := `event: unleash-updated
data: {"events":[{"type":"feature-updated","eventId":2,"feature":{"name":"test2","enabled":true,"strategies":[]}}]}

`
			w.Write([]byte(update))
			flusher.Flush()
			
			// Keep connection alive
			time.Sleep(500 * time.Millisecond)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create error listener to capture errors
	errorCount := int32(0)
	listener := &MockedListener{}
	listener.On("OnError", mock.Anything).Run(func(args mock.Arguments) {
		atomic.AddInt32(&errorCount, 1)
		t.Logf("Error received: %v", args.Get(0))
	}).Maybe()
	listener.On("OnWarning", mock.Anything).Maybe()
	listener.On("OnReady").Maybe()
	listener.On("OnUpdate").Maybe()

	client, err := NewClient(
		WithUrl(server.URL),
		WithAppName("test-app"),
		WithInstanceId("test-instance"),
		WithDisableMetrics(true),
		WithExperimentalMode(map[string]string{"type": "streaming"}),
		WithStorage(NewDefaultDeltaStorage()),
		WithListener(listener),
	)
	assert.NoError(t, err)
	defer client.Close()

	// Wait for client to be ready
	client.WaitForReady()
	
	// First feature should be available
	assert.True(t, client.IsEnabled("test"), "test feature should be enabled")
	
	// Wait for updates to be processed
	time.Sleep(500 * time.Millisecond)
	
	// Second feature should also be available (malformed event was skipped)
	assert.True(t, client.IsEnabled("test2"), "test2 feature should be enabled after valid update")
	
	// Should have received at least one error for the malformed event
	assert.Greater(t, atomic.LoadInt32(&errorCount), int32(0), 
		"Should have logged error for malformed event")
}