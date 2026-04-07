package unleash

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/launchdarkly/eventsource"
)

type testEvent struct {
	event string
	data  string
}

func (e testEvent) Id() string    { return "" }
func (e testEvent) Event() string { return e.event }
func (e testEvent) Data() string  { return e.data }

func TestStreamingFetcher_BrokenEventRestartsStream(t *testing.T) {
	baseURL, err := url.Parse("http://example.com/")
	if err != nil {
		t.Fatalf("failed to parse base URL: %v", err)
	}

	channels := makeBaseChannels()
	fetcher := newStreamingFetcher(fetcherOptions{
		appName:    "test-app",
		instanceId: "test-instance",
		url:        *baseURL,
		storage:    &NoOpStorage{},
		httpClient: http.DefaultClient,
		headers:    http.Header{},
	}, channels, nil)

	stream := &eventsource.Stream{
		Events: make(chan eventsource.Event, 1),
	}

	fetcher.subscribe = func(*http.Request, ...eventsource.StreamOption) (*eventsource.Stream, error) {
		return stream, nil
	}

	restarts := make(chan struct{}, 1)
	fetcher.restartStreamFn = func() {
		restarts <- struct{}{}
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/stream", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	go fetcher.runStream(context.Background(), req)

	stream.Events <- testEvent{
		event: "unleash-updated",
		data:  "should-be-json-whee",
	}
	close(stream.Events)

	select {
	case <-restarts:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected stream restart after broken event")
	}
}
