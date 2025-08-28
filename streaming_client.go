package unleash

import (
	"context"
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

	sc.processor = newStreamingProcessor(storage, sc.repository, sc.repositoryChannels)

	log.Print("Setting up client")

	req, err := http.NewRequestWithContext(sc.ctx, "GET", sc.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, values := range sc.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	
	req.Header.Set("UNLEASH-APPNAME", sc.appName)
	req.Header.Set("UNLEASH-INSTANCEID", sc.instanceId)
	req.Header.Add("Unleash-Client-Spec", SEGMENT_CLIENT_SPEC_VERSION)

	// Retry initial connection attempts with configurable duration and exponential backoff
	retryDuration := sc.repository.options.streamingRetryDuration
	if retryDuration <= 0 {
		retryDuration = 5 * time.Minute // fallback default
	}
	startTime := time.Now()
	baseDelay := time.Second
	maxDelay := 30 * time.Second // Cap individual delays at 30 seconds
	
	var stream *eventsource.Stream
	var streamErr error
	attempt := 0
	
	for time.Since(startTime) < retryDuration {
		stream, streamErr = eventsource.SubscribeWithRequestAndOptions(req,
			eventsource.StreamOptionHTTPClient(sc.httpClient),
			eventsource.StreamOptionUseBackoff(5*time.Minute),
			eventsource.StreamOptionUseJitter(0.5),
			eventsource.StreamOptionErrorHandler(func(err error) eventsource.StreamErrorHandlerResult {
				sc.errorChannels.err(fmt.Errorf("SSE error: %w", err))
				return eventsource.StreamErrorHandlerResult{CloseNow: false}
			}),
		)
		
		if streamErr == nil {
			// Connection successful
			break
		}
		
		attempt++
		elapsed := time.Since(startTime)
		remaining := retryDuration - elapsed
		
		// Log the attempt with timing info
		sc.errorChannels.err(fmt.Errorf("streaming connection attempt %d failed (elapsed: %v, remaining: %v): %w", 
			attempt, elapsed.Round(time.Second), remaining.Round(time.Second), streamErr))
		
		// If we still have time left, calculate backoff delay
		if remaining > 0 {
			// Exponential backoff: min(baseDelay * 2^attempt, maxDelay)
			backoffDelay := baseDelay * time.Duration(1<<uint(attempt-1))
			if backoffDelay > maxDelay {
				backoffDelay = maxDelay
			}
			
			// Don't wait longer than remaining time
			if backoffDelay > remaining {
				backoffDelay = remaining
			}
			
			if backoffDelay > 0 {
				time.Sleep(backoffDelay)
			}
		}
	}
	
	if streamErr != nil {
		return fmt.Errorf("failed to establish streaming connection after %v (attempted %d times): %w", 
			retryDuration, attempt, streamErr)
	}

	sc.stream = stream

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
		sc.errorChannels.warn(fmt.Errorf("unknown SSE event type: %s", eventType))
	}
}

// handleConnectedEvent processes the initial connection event
func (sc *streamingClient) handleConnectedEvent(event eventsource.Event) error {
	eventData := []byte(event.Data())
	delta, err := api.ParseDelta(eventData)
	if err != nil {
		return fmt.Errorf("failed to parse connected event: %w", err)
	}
	
	return sc.processor.processDelta(delta)
}

// handleUpdatedEvent processes feature update events
func (sc *streamingClient) handleUpdatedEvent(event eventsource.Event) error {
	eventData := []byte(event.Data())
	delta, err := api.ParseDelta(eventData)
	if err != nil {
		return fmt.Errorf("failed to parse updated event: %w", err)
	}
	
	return sc.processor.processDelta(delta)
}

// stop closes the SSE connection
func (sc *streamingClient) stop() {
	sc.runningMutex.Lock()
	defer sc.runningMutex.Unlock()

	if !sc.running {
		return
	}

	sc.cancel()
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