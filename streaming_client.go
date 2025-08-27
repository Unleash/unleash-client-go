package unleash

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5/api"
	"github.com/launchdarkly/eventsource"
)

// streamingClient handles the SSE connection for streaming feature updates
type streamingClient struct {
	url                string
	appName            string
	instanceId         string
	httpClient         *http.Client
	headers            http.Header
	stream             *eventsource.Stream
	processor          *streamingProcessor
	repository         *repository
	ctx                context.Context
	cancel             context.CancelFunc
	running            bool
	runningMutex       sync.RWMutex
	errorChannels      errorChannels
	repositoryChannels repositoryChannels
}

// newStreamingClient creates a new streaming client
func newStreamingClient(options repositoryOptions, repo *repository, repoChannels repositoryChannels, errChannels errorChannels) *streamingClient {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &streamingClient{
		url:                fmt.Sprintf("%sclient/streaming", options.url.String()),
		appName:            options.appName,
		instanceId:         options.instanceId,
		httpClient:         options.httpClient,
		headers:            options.headers,
		repository:         repo,
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

	// Initialize the processor with the storage and repository
	sc.processor = newStreamingProcessor(storage, sc.repository, sc.repositoryChannels)

	log.Print("Setting up client")

	// Create HTTP request with custom headers
	req, err := http.NewRequestWithContext(sc.ctx, "GET", sc.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Copy custom headers
	for key, values := range sc.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	
	// Add Unleash-specific headers
	req.Header.Set("UNLEASH-APPNAME", sc.appName)
	req.Header.Set("UNLEASH-INSTANCEID", sc.instanceId)

	// Create eventsource stream with options
	stream, err := eventsource.SubscribeWithRequestAndOptions(req,
		eventsource.StreamOptionReadTimeout(30*time.Second),
		eventsource.StreamOptionUseBackoff(5*time.Minute),
		eventsource.StreamOptionUseJitter(0.5),
		eventsource.StreamOptionErrorHandler(func(err error) eventsource.StreamErrorHandlerResult {
			sc.errorChannels.err(fmt.Errorf("SSE error: %w", err))
			return eventsource.StreamErrorHandlerResult{CloseNow: false}
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to subscribe to event stream: %w", err)
	}

	sc.stream = stream

	// Handle events in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("SSE subscription panic recovered: %v", r)
				sc.errorChannels.err(err)
			}
		}()
		
		for {
			select {
			case event := <-stream.Events:
				if event != nil {
					sc.handleEvent(event)
				}
			case <-sc.ctx.Done():
				stream.Close()
				return
			}
		}
	}()

	sc.running = true
	return nil
}

// handleEvent processes individual SSE events
func (sc *streamingClient) handleEvent(event eventsource.Event) {
	eventType := event.Event()
	log.Printf("Handling event: %s", eventType)

	if event == nil {
		return
	}

	log.Printf("Received SSE event: %s", eventType)
	log.Printf("Receiving SSE content %s", event.Data())

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
func (sc *streamingClient) handleConnectedEvent(event eventsource.Event) error {
	// Try to parse as delta format first
	eventData := []byte(event.Data())
	delta, err := api.ParseDelta(eventData)
	if err == nil && len(delta.Events) > 0 {
		// Process as delta event
		return sc.processor.processDelta(delta)
	}
	
	// Fallback to legacy format for backward compatibility
	var eventDataMap map[string]interface{}
	if err := json.Unmarshal(eventData, &eventDataMap); err != nil {
		return fmt.Errorf("failed to parse connected event: %w", err)
	}

	// Check if this is a delta format with events array
	if events, ok := eventDataMap["events"]; ok && events != nil {
		// This is delta format - parse and process it
		delta, err := api.ParseDelta(eventData)
		if err != nil {
			return fmt.Errorf("failed to parse delta event: %w", err)
		}
		return sc.processor.processDelta(delta)
	}

	// Legacy format: Extract features from the event data
	if features, ok := eventDataMap["features"]; ok {
		// Convert to FeatureResponse format
		featuresJSON, err := json.Marshal(map[string]interface{}{
			"features": features,
			"segments": eventDataMap["segments"],
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
func (sc *streamingClient) handleUpdatedEvent(event eventsource.Event) error {
	// Try to parse as delta format first
	eventData := []byte(event.Data())
	delta, err := api.ParseDelta(eventData)
	if err == nil && len(delta.Events) > 0 {
		// Process as delta event
		return sc.processor.processDelta(delta)
	}
	
	// Fallback to legacy format for backward compatibility
	var eventDataMap map[string]interface{}
	if err := json.Unmarshal(eventData, &eventDataMap); err != nil {
		return fmt.Errorf("failed to parse updated event: %w", err)
	}

	// Check if this is a delta format with events array
	if events, ok := eventDataMap["events"]; ok && events != nil {
		// This is delta format - parse and process it
		delta, err := api.ParseDelta(eventData)
		if err != nil {
			return fmt.Errorf("failed to parse delta event: %w", err)
		}
		return sc.processor.processDelta(delta)
	}

	// Legacy format: Extract features from the event data
	if features, ok := eventDataMap["features"]; ok {
		// Convert to FeatureResponse format
		featuresJSON, err := json.Marshal(map[string]interface{}{
			"features": features,
			"segments": eventDataMap["segments"],
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
	// Close the eventsource stream
	if sc.stream != nil {
		sc.stream.Close()
	}
	sc.running = false
}

// isRunning returns whether the streaming client is currently running
func (sc *streamingClient) isRunning() bool {
	sc.runningMutex.RLock()
	defer sc.runningMutex.RUnlock()
	return sc.running
}