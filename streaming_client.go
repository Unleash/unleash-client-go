package unleash

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/launchdarkly/eventsource"
)

type streamErrorKind int

const (
	transient streamErrorKind = iota
	warning
	fatal
	corrupted
)

type streamError struct {
	kind streamErrorKind
	err  error
}

type streamingClient struct {
<<<<<<< Updated upstream
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
=======
	repositoryChannels
	url            string
	appName        string
	instanceId     string
	httpClient     *http.Client
	headers        http.Header
	stream         *eventsource.Stream
	storage        Storage
	deltaProcessor *deltaProcessor
	ctx            context.Context
	cancel         context.CancelFunc
	readyOnce      sync.Once
	internalErrors chan streamError
>>>>>>> Stashed changes
}

func newStreamingClient(options repositoryOptions, repoChannels repositoryChannels) *streamingClient {
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
		internalErrors:     make(chan streamError, 8),
		storage:            options.storage,
	}

	streamingFetcher.sync()

	return streamingFetcher
}

<<<<<<< Updated upstream
func (sc *streamingClient) sync() error {

=======
func (sc *streamingClient) start() {
>>>>>>> Stashed changes
	req, err := http.NewRequestWithContext(sc.ctx, "GET", sc.url, nil)
	if err != nil {
		sc.internalErrors <- streamError{kind: fatal, err: fmt.Errorf("failed to create request: %w", err)}
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

	go sc.runStream(req)
	go sc.superviseStream()
}

func (sc *streamingClient) runStream(req *http.Request) {
	stream, err := eventsource.SubscribeWithRequestAndOptions(req,
		eventsource.StreamOptionCanRetryFirstConnection(-time.Second*3),
		eventsource.StreamOptionUseBackoff(5*time.Minute),
		eventsource.StreamOptionUseJitter(0.5),
		eventsource.StreamOptionErrorHandler(func(err error) eventsource.StreamErrorHandlerResult {
			sc.internalErrors <- streamError{kind: transient, err: fmt.Errorf("SSE error: %w", err)}
			return eventsource.StreamErrorHandlerResult{CloseNow: false}
		}),
	)

	if err != nil {
		sc.internalErrors <- streamError{kind: fatal, err: fmt.Errorf("failed to subscribe to SSE stream: %w", err)}
		return
	}

	sc.stream = stream

	for {
		select {
		case event, ok := <-stream.Events:
			if !ok {
				return
			}
			switch event.Event() {
			case "unleash-connected", "unleash-updated":
				if err := sc.handleDomainEvent(event); err != nil {
					sc.internalErrors <- streamError{kind: corrupted, err: fmt.Errorf("failed to handle event: %w", err)}
				}
			}
		case <-sc.ctx.Done():
			stream.Close()
			return
		}
	}
}

func (sc *streamingClient) superviseStream() {
	// this intentionally just black holes errors. This is incomplete
	// in this PR and will be fleshed out in the next steps
	for {
		select {
		case _, ok := <-sc.internalErrors:
			if !ok {
				return
			}

		case <-sc.ctx.Done():
			return
		}
	}
}

<<<<<<< Updated upstream
func (sc *streamingClient) handleEvent(event eventsource.Event) {
	if event == nil {
		return
	}

	eventType := event.Event()

	switch eventType {
	case "unleash-connected", "unleash-updated":
		if err := sc.handleDomainEvent(event); err != nil {
			// need to handle failover here but in a future PR
			// this absolutely cannot just log. Something has gone
			// very badly wrong and continuing leads to a corrupted state
		}
	}
=======
func (sc *streamingClient) markReady() {
	sc.readyOnce.Do(func() {
		sc.ready <- true
	})
>>>>>>> Stashed changes
}

func (sc *streamingClient) handleDomainEvent(event eventsource.Event) error {
	eventData := []byte(event.Data())
	delta, err := api.ParseDelta(eventData)
	if err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}

<<<<<<< Updated upstream
	return sc.deltaProcessor.process(delta)
=======
	backupState, err := sc.deltaProcessor.process(delta)

	if err != nil {
		return fmt.Errorf("failed to process delta: %w", err)
	}

	sc.markReady()

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
>>>>>>> Stashed changes
}

func (sc *streamingClient) snapshot() *FeatureMemoryState {
	return sc.deltaProcessor.snapshot()
}

func (sc *streamingClient) stop() {
	sc.cancel()
}
