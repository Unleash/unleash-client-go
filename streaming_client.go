package unleash

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/Unleash/unleash-go-sdk/v5/api"
	"github.com/r3labs/sse/v2"
)

// streamingClient handles the SSE connection for streaming feature updates
type streamingClient struct {
	url            string
	appName        string
	instanceId     string
	httpClient     *http.Client
	headers        http.Header
	client         *sse.Client
	processor      *streamingProcessor
	ctx            context.Context
	cancel         context.CancelFunc
	running        bool
	runningMutex   sync.RWMutex
	errorChannels  errorChannels
	repositoryChannels repositoryChannels
}

// newStreamingClient creates a new streaming client
func newStreamingClient(options repositoryOptions, repoChannels repositoryChannels, errChannels errorChannels) *streamingClient {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &streamingClient{
		url:                fmt.Sprintf("%sclient/streaming", options.url.String()),
		appName:            options.appName,
		instanceId:         options.instanceId,
		httpClient:         options.httpClient,
		headers:            options.headers,
		ctx:                ctx,
		cancel:             cancel,
		errorChannels:      errChannels,
		repositoryChannels: repoChannels,
		running:            false,
	}
}

// start begins the SSE connection
func (sc *streamingClient) start(storage Storage) error {
	sc.runningMutex.Lock()
	defer sc.runningMutex.Unlock()

	if sc.running {
		return nil
	}

	// Initialize the processor with the storage
	sc.processor = newStreamingProcessor(storage, sc.repositoryChannels)

	// Create SSE client with custom headers
	client := sse.NewClient(sc.url)
	
	// Convert http.Header to map[string]string
	if client.Headers == nil {
		client.Headers = make(map[string]string)
	}
	
	// Copy custom headers
	for key, values := range sc.headers {
		if len(values) > 0 {
			client.Headers[key] = values[0] // Take the first value
		}
	}
	
	// Add Unleash-specific headers
	client.Headers["UNLEASH-APPNAME"] = sc.appName
	client.Headers["UNLEASH-INSTANCEID"] = sc.instanceId

	sc.client = client

	// Subscribe to SSE events using handler function
	go func() {
		err := client.SubscribeWithContext(sc.ctx, "", sc.handleEvent)
		if err != nil {
			sc.errorChannels.err(fmt.Errorf("SSE subscription error: %w", err))
		}
	}()

	sc.running = true
	return nil
}

// handleEvent processes individual SSE events
func (sc *streamingClient) handleEvent(event *sse.Event) {
	if event == nil {
		return
	}

	eventType := string(event.Event)
	switch eventType {
	case "unleash-connected":
		if err := sc.handleConnectedEvent(event); err != nil {
			sc.errorChannels.err(fmt.Errorf("error handling connected event: %w", err))
		}
	case "unleash-updated":
		if err := sc.handleUpdatedEvent(event); err != nil {
			sc.errorChannels.err(fmt.Errorf("error handling updated event: %w", err))
		}
	default:
		// Unknown event type, log as warning
		sc.errorChannels.warn(fmt.Errorf("unknown SSE event type: %s", eventType))
	}
}

// handleConnectedEvent processes the initial connection event
func (sc *streamingClient) handleConnectedEvent(event *sse.Event) error {
	// Try to parse as delta format first
	delta, err := api.ParseDelta(event.Data)
	if err == nil && len(delta.Events) > 0 {
		// Process as delta event
		return sc.processor.processDelta(delta)
	}
	
	// Fallback to legacy format for backward compatibility
	var eventData map[string]interface{}
	if err := json.Unmarshal(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to parse connected event: %w", err)
	}

	// Check if this is a delta format with events array
	if events, ok := eventData["events"]; ok && events != nil {
		// This is delta format - parse and process it
		delta, err := api.ParseDelta(event.Data)
		if err != nil {
			return fmt.Errorf("failed to parse delta event: %w", err)
		}
		return sc.processor.processDelta(delta)
	}

	// Legacy format: Extract features from the event data
	if features, ok := eventData["features"]; ok {
		// Convert to FeatureResponse format
		featuresJSON, err := json.Marshal(map[string]interface{}{
			"features": features,
			"segments": eventData["segments"],
		})
		if err != nil {
			return err
		}

		var featureResp api.FeatureResponse
		if err := json.Unmarshal(featuresJSON, &featureResp); err != nil {
			return err
		}

		// Process the initial feature set
		if err := sc.processor.processFeatureResponse(featureResp); err != nil {
			return err
		}
	}

	return nil
}

// handleUpdatedEvent processes feature update events
func (sc *streamingClient) handleUpdatedEvent(event *sse.Event) error {
	// Try to parse as delta format first
	delta, err := api.ParseDelta(event.Data)
	if err == nil && len(delta.Events) > 0 {
		// Process as delta event
		return sc.processor.processDelta(delta)
	}
	
	// Fallback to legacy format for backward compatibility
	var eventData map[string]interface{}
	if err := json.Unmarshal(event.Data, &eventData); err != nil {
		return fmt.Errorf("failed to parse updated event: %w", err)
	}

	// Check if this is a delta format with events array
	if events, ok := eventData["events"]; ok && events != nil {
		// This is delta format - parse and process it
		delta, err := api.ParseDelta(event.Data)
		if err != nil {
			return fmt.Errorf("failed to parse delta event: %w", err)
		}
		return sc.processor.processDelta(delta)
	}

	// Legacy format: Extract features from the event data
	if features, ok := eventData["features"]; ok {
		// Convert to FeatureResponse format
		featuresJSON, err := json.Marshal(map[string]interface{}{
			"features": features,
			"segments": eventData["segments"],
		})
		if err != nil {
			return err
		}

		var featureResp api.FeatureResponse
		if err := json.Unmarshal(featuresJSON, &featureResp); err != nil {
			return err
		}

		// Process the feature updates
		if err := sc.processor.processFeatureResponse(featureResp); err != nil {
			return err
		}
	}

	return nil
}

// stop closes the SSE connection
func (sc *streamingClient) stop() {
	sc.runningMutex.Lock()
	defer sc.runningMutex.Unlock()

	if !sc.running {
		return
	}

	sc.cancel()
	// The SSE client will automatically close when context is cancelled
	sc.running = false
}

// isRunning returns whether the streaming client is currently running
func (sc *streamingClient) isRunning() bool {
	sc.runningMutex.RLock()
	defer sc.runningMutex.RUnlock()
	return sc.running
}