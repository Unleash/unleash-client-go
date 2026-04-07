package unleash

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/launchdarkly/eventsource"
)

type streamingFetcher struct {
	fetcherChannels
	url            string
	appName        string
	instanceId     string
	httpClient     *http.Client
	headers        http.Header
	storage        Storage
	deltaProcessor *deltaProcessor

	ctx    context.Context
	cancel context.CancelFunc

	streamMu     sync.Mutex
	streamCancel context.CancelFunc

	hydrated        atomic.Bool
	recordFailEvent func(failEvent)
	subscribe       func(*http.Request, ...eventsource.StreamOption) (*eventsource.Stream, error)
	restartStreamFn func()
}

func buildFailoverRecorder(ch chan failEvent, failoverStrategy *failoverStrategy) func(failEvent) {
	if ch == nil {
		return func(failEvent) {}
	}

	var once sync.Once
	return func(reason failEvent) {
		if failoverStrategy.shouldFailover(reason, time.Now()) {
			once.Do(func() {
				ch <- reason
			})
		}
	}
}

func newStreamingFetcher(options fetcherOptions, fetcherChannels fetcherChannels, failoverChannel chan failEvent) *streamingFetcher {
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

	streamingFetcher := &streamingFetcher{
		url:             fmt.Sprintf("%sclient/streaming", options.url.String()),
		appName:         options.appName,
		instanceId:      options.instanceId,
		httpClient:      options.httpClient,
		headers:         options.headers,
		deltaProcessor:  deltaProc,
		ctx:             ctx,
		cancel:          cancel,
		fetcherChannels: fetcherChannels,
		recordFailEvent: buildFailoverRecorder(failoverChannel, newFailoverStrategy(5, 60*time.Second)),
		storage:         options.storage,
	}
	streamingFetcher.subscribe = eventsource.SubscribeWithRequestAndOptions
	streamingFetcher.restartStreamFn = streamingFetcher.restartStream

	return streamingFetcher
}

func (sc *streamingFetcher) start() {
	sc.restartStream()
}

func (sc *streamingFetcher) restartStream() {
	sc.streamMu.Lock()
	defer sc.streamMu.Unlock()

	if sc.ctx.Err() != nil {
		return
	}

	if sc.streamCancel != nil {
		sc.streamCancel()
	}

	streamCtx, streamCancel := context.WithCancel(sc.ctx)
	sc.streamCancel = streamCancel

	req, err := http.NewRequestWithContext(streamCtx, "GET", sc.url, nil)
	if err != nil {
		sc.recordFailEvent(&failoverRequest{
			baseFailEvent: baseFailEvent{
				occurredAt: time.Now(),
				message:    fmt.Sprintf("unable to setup base http request for streaming fetcher: %v", err),
			},
		})
		return
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

	go sc.runStream(streamCtx, req)
}

func (sc *streamingFetcher) runStream(streamCtx context.Context, req *http.Request) {
	stream, err := sc.subscribe(req,
		eventsource.StreamOptionCanRetryFirstConnection(-time.Second*3),
		eventsource.StreamOptionUseBackoff(5*time.Minute),
		eventsource.StreamOptionUseJitter(0.5),
		eventsource.StreamOptionErrorHandler(func(err error) eventsource.StreamErrorHandlerResult {

			var event failEvent

			if se, ok := err.(eventsource.SubscriptionError); ok {
				status := se.Code
				event = &httpStatusErrorEvent{
					baseFailEvent: baseFailEvent{
						occurredAt: time.Now(),
						message:    fmt.Sprintf("streaming connection error with HTTP status code %d: %v", status, err),
					},
					statusCode: status,
				}
			} else {
				event = &networkErrorEvent{
					baseFailEvent: baseFailEvent{
						occurredAt: time.Now(),
						message:    fmt.Sprintf("streaming connection network error: %v", err),
					},
					err: err,
				}
			}
			sc.recordFailEvent(event)

			// Don't need to close, this is the job of of the wrapping adaptiveFetcher to decide when
			// or if to cutover. We just let it know and if it wants to close it'll invoke stop
			return eventsource.StreamErrorHandlerResult{CloseNow: false}
		}),
	)

	if err != nil {
		sc.recordFailEvent(&failoverRequest{baseFailEvent: baseFailEvent{occurredAt: time.Now(), message: fmt.Sprintf("failed to establish streaming connection: %v", err)}})
		return
	}

	for {
		select {
		case event, ok := <-stream.Events:
			if !ok {
				return
			}
			switch event.Event() {
			case "unleash-connected", "unleash-updated":
				if err := sc.handleDomainEvent(event); err != nil {
					go sc.restartStreamFn()
				}
			}
		case <-streamCtx.Done():
			stream.Close()
			return
		}
	}
}

func (sc *streamingFetcher) notifyUpdate() {
	if sc.hydrated.CompareAndSwap(false, true) {
		sc.ready <- true
	} else {
		sc.update <- true
	}
}

func (sc *streamingFetcher) handleDomainEvent(event eventsource.Event) error {
	eventData := []byte(event.Data())
	delta, err := api.ParseDelta(eventData)
	if err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

	backupState, err := sc.deltaProcessor.process(delta)

	if err != nil {
		return fmt.Errorf("failed to process delta: %w", err)
	}

	sc.notifyUpdate()

	// Failure to save a backup is definitely not an error but end users should have
	// a way of detecting that this isn't working correctly so we throw it on the warnings channel
	// this just bypasses the internal error handling mechanisms because this never needs to inform
	// failover or reconnection strategies
	if err := sc.storage.Persist(backupState); err != nil {
		select {
		case sc.warnings <- fmt.Errorf("failed to persist state: %w", err):
		default:
		}
	}

	return nil
}

func (sc *streamingFetcher) snapshot() *FeatureMemoryState {
	return sc.deltaProcessor.snapshot()
}

func (sc *streamingFetcher) stop() {
	sc.cancel()

	sc.streamMu.Lock()
	defer sc.streamMu.Unlock()
	
	if sc.streamCancel != nil {
		sc.streamCancel()
	}
}
