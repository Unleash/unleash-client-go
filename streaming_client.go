package unleash

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/launchdarkly/eventsource"
)

type streamingClient struct {
	url                string
	appName            string
	instanceId         string
	httpClient         *http.Client
	headers            http.Header
	stream             *eventsource.Stream
	storage            Storage
	deltaProcessor     *deltaProcessor
	ctx                context.Context
	cancel             context.CancelFunc
	repositoryChannels repositoryChannels
}

func newStreamingFetcher(options repositoryOptions, repoChannels repositoryChannels) *streamingClient {
	ctx, cancel := context.WithCancel(context.Background())

	var apiResponse *api.FeatureResponse

	if loadedState, err := options.storage.Load(); err == nil && loadedState != nil {
		apiResponse = loadedState
	} else {
		apiResponse = &api.FeatureResponse{
			Features: []api.Feature{},
			Segments: []api.Segment{},
		}
	}

	deltaProc := newDeltaProcessor(apiResponse)

	streamingFetcher := &streamingClient{
		url:                fmt.Sprintf("%sclient/streaming", options.url.String()),
		appName:            options.appName,
		instanceId:         options.instanceId,
		httpClient:         options.httpClient,
		headers:            options.headers,
		deltaProcessor:     deltaProc,
		ctx:                ctx,
		cancel:             cancel,
		repositoryChannels: repoChannels,
	}

	streamingFetcher.sync()

	return streamingFetcher
}

func (sc *streamingClient) sync() error {

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

	return nil
}

func (sf *streamingClient) handleEvent(event eventsource.Event) {
	if event == nil {
		return
	}

	eventType := event.Event()

	switch eventType {
	case "unleash-connected", "unleash-updated":
		if err := sf.handleDomainEvent(event); err != nil {
			// need to handle failover here but in a future PR
			// this absolutely cannot just log. Something has gone
			// very badly wrong and continuing leads to a corrupted state
		}
	}
}

func (sf *streamingClient) handleDomainEvent(event eventsource.Event) error {
	eventData := []byte(event.Data())
	delta, err := api.ParseDelta(eventData)
	if err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	return sf.deltaProcessor.process(delta)
}

func (sf *streamingClient) snapshot() *FeatureMemoryState {
	return sf.deltaProcessor.snapshot()
}

func (sf *streamingClient) Close() {
	sf.cancel()
	if sf.stream != nil {
		sf.stream.Close()
	}
}
