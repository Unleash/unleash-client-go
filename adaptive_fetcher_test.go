package unleash

import (
	"testing"

	"github.com/Unleash/unleash-go-sdk/v6/api"
)

type fakeFetcher struct {
	startCalls int
	stopCalls  int
	state      *FeatureMemoryState
}

func (f *fakeFetcher) start() {
	f.startCalls++
}

func (f *fakeFetcher) stop() {
	f.stopCalls++
}

func (f *fakeFetcher) snapshot() *FeatureMemoryState {
	return f.state
}

func makeBaseChannels() fetcherChannels {
	return fetcherChannels{
		errorChannels: errorChannels{
			errors:   make(chan error, 1),
			warnings: make(chan error, 1),
		},
		ready:  make(chan bool, 1),
		update: make(chan bool, 1),
	}
}

func makeTestFactory(streamingFetcher *fakeFetcher, pollingFetcher *fakeFetcher) fetcherFactory {
	return fetcherFactory{
		newStreaming: func(options fetcherOptions, channels fetcherChannels, streamErrorChannel chan streamError) togglerFetcher {
			return streamingFetcher
		},
		newPolling: func(options fetcherOptions, channels fetcherChannels) togglerFetcher {
			return pollingFetcher
		},
	}
}

func TestAdaptiveFetcher_CutoverSwapsToPolling(t *testing.T) {
	streaming := &fakeFetcher{
		state: &FeatureMemoryState{
			Features: map[string]*api.Feature{"streaming": {}},
			Segments: map[int][]api.Constraint{},
		},
	}

	polling := &fakeFetcher{
		state: &FeatureMemoryState{
			Features: map[string]*api.Feature{"polling": {}},
			Segments: map[int][]api.Constraint{},
		},
	}

	factory := makeTestFactory(streaming, polling)
	channels := makeBaseChannels()

	af := newAdaptiveFetcherWithFactory(fetcherOptions{}, channels, factory)

	af.start()
	if streaming.startCalls != 1 {
		t.Fatalf("expected streaming fetcher to start once, got %d", streaming.startCalls)
	}

	af.cutover()

	if streaming.stopCalls != 1 {
		t.Fatalf("expected streaming fetcher to stop once, got %d", streaming.stopCalls)
	}

	if polling.startCalls != 1 {
		t.Fatalf("expected polling fetcher to start once, got %d", polling.startCalls)
	}

	got := af.snapshot()
	if _, ok := got.Features["polling"]; !ok {
		t.Fatalf("expected snapshot from polling fetcher after cutover")
	}
}

func TestAdaptiveFetcher_StartIsIdempotent(t *testing.T) {
	streaming := &fakeFetcher{
		state: &FeatureMemoryState{
			Features: map[string]*api.Feature{"streaming": {}},
			Segments: map[int][]api.Constraint{},
		},
	}

	factory := makeTestFactory(streaming, nil)
	channels := makeBaseChannels()

	af := newAdaptiveFetcherWithFactory(fetcherOptions{}, channels, factory)

	af.start()
	if streaming.startCalls != 1 {
		t.Fatalf("expected streaming fetcher to start once, got %d", streaming.startCalls)
	}

	af.start()
	if streaming.startCalls != 1 {
		t.Fatalf("expected streaming fetcher to not start again, got %d", streaming.startCalls)
	}
}

func TestAdaptiveFetcher_StopIsIdempotent(t *testing.T) {
	streaming := &fakeFetcher{
		state: &FeatureMemoryState{
			Features: map[string]*api.Feature{"streaming": {}},
			Segments: map[int][]api.Constraint{},
		},
	}

	factory := makeTestFactory(streaming, nil)
	channels := makeBaseChannels()

	af := newAdaptiveFetcherWithFactory(fetcherOptions{}, channels, factory)

	af.start()
	af.stop()
	if streaming.stopCalls != 1 {
		t.Fatalf("expected streaming fetcher to stop once, got %d", streaming.stopCalls)
	}

	af.stop()
	if streaming.stopCalls != 1 {
		t.Fatalf("expected streaming fetcher to not stop again, got %d", streaming.stopCalls)
	}
}

func TestAdaptiveFetcher_SnapshotDelegatesToCurrentFetcher(t *testing.T) {
	streaming := &fakeFetcher{
		state: &FeatureMemoryState{
			Features: map[string]*api.Feature{"streaming": {}},
			Segments: map[int][]api.Constraint{},
		},
	}

	factory := makeTestFactory(streaming, nil)
	channels := makeBaseChannels()

	af := newAdaptiveFetcherWithFactory(fetcherOptions{}, channels, factory)

	got := af.snapshot()
	if _, ok := got.Features["streaming"]; !ok {
		t.Fatalf("expected snapshot from streaming fetcher")
	}
}

func TestAdaptiveFetcher_SnapshotAfterCutover(t *testing.T) {
	streaming := &fakeFetcher{
		state: &FeatureMemoryState{
			Features: map[string]*api.Feature{"streaming": {}},
			Segments: map[int][]api.Constraint{},
		},
	}

	polling := &fakeFetcher{
		state: &FeatureMemoryState{
			Features: map[string]*api.Feature{"polling": {}},
			Segments: map[int][]api.Constraint{},
		},
	}

	factory := makeTestFactory(streaming, polling)
	channels := makeBaseChannels()

	af := newAdaptiveFetcherWithFactory(fetcherOptions{}, channels, factory)

	af.start()
	af.cutover()

	got := af.snapshot()
	if _, ok := got.Features["polling"]; !ok {
		t.Fatalf("expected snapshot from polling fetcher after cutover")
	}
}
