package unleash

import (
	"context"
	"fmt"
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
	deltaProcessor     *deltaProcessor
	ctx                context.Context
	cancel             context.CancelFunc
	running            bool
	runningMutex       sync.RWMutex
	repositoryChannels repositoryChannels
}

func newStreamingClient(options repositoryOptions, repoChannels repositoryChannels, deltaProc *deltaProcessor) *streamingClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &streamingClient{
		url:                fmt.Sprintf("%sclient/streaming", options.url.String()),
		appName:            options.appName,
		instanceId:         options.instanceId,
		httpClient:         options.httpClient,
		headers:            options.headers,
		deltaProcessor:     deltaProc,
		ctx:                ctx,
		cancel:             cancel,
		repositoryChannels: repoChannels,
		running:            false,
	}
}

func (sc *streamingClient) start(_ Storage) error {
	sc.runningMutex.Lock()
	defer sc.runningMutex.Unlock()

	if sc.running {
		return nil
	}

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
	req.Header.Add("User-Agent", sc.appName)

	stream, err := eventsource.SubscribeWithRequestAndOptions(req,
		eventsource.StreamOptionCanRetryFirstConnection(-time.Second*3),
		eventsource.StreamOptionUseBackoff(5*time.Minute),
		eventsource.StreamOptionUseJitter(0.5),
		eventsource.StreamOptionErrorHandler(func(err error) eventsource.StreamErrorHandlerResult {
			sc.repositoryChannels.errorChannels.err(fmt.Errorf("SSE error: %w", err))
			return eventsource.StreamErrorHandlerResult{CloseNow: false}
		}),
	)

	if err != nil {
		return fmt.Errorf("failed to establish streaming connection: %w", err)
	}

	sc.stream = stream

	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("SSE subscription panic recovered: %v", r)
				sc.repositoryChannels.errorChannels.err(err)
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

func (sc *streamingClient) handleEvent(event eventsource.Event) {
	eventType := event.Event()

	if event == nil {
		return
	}

	switch eventType {
	case "unleash-connected":
		if err := sc.handleConnectedEvent(event); err != nil {
			sc.repositoryChannels.errorChannels.err(fmt.Errorf("error handling connected event: %w", err))
		}
	case "unleash-updated":
		if err := sc.handleUpdatedEvent(event); err != nil {
			sc.repositoryChannels.errorChannels.err(fmt.Errorf("error handling updated event: %w", err))
		}
	default:
		sc.repositoryChannels.errorChannels.warn(fmt.Errorf("unknown SSE event type: %s", eventType))
	}
}

func (sc *streamingClient) handleConnectedEvent(event eventsource.Event) error {
	eventData := []byte(event.Data())
	delta, err := api.ParseDelta(eventData)
	if err != nil {
		return fmt.Errorf("failed to parse connected event: %w", err)
	}

	return sc.deltaProcessor.process(delta)
}

func (sc *streamingClient) handleUpdatedEvent(event eventsource.Event) error {
	eventData := []byte(event.Data())
	delta, err := api.ParseDelta(eventData)
	if err != nil {
		return fmt.Errorf("failed to parse updated event: %w", err)
	}

	return sc.deltaProcessor.process(delta)
}

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

func (sc *streamingClient) isRunning() bool {
	sc.runningMutex.RLock()
	defer sc.runningMutex.RUnlock()
	return sc.running
}
