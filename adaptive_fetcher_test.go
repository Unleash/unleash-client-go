package unleash

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/Unleash/unleash-go-sdk/v6/api"
)

type fakeFetcher struct {
	startCalls atomic.Int32
	stopCalls  atomic.Int32
	state      *FeatureMemoryState
}

func (f *fakeFetcher) start() {
	f.startCalls.Add(1)
}

func (f *fakeFetcher) stop() {
	f.stopCalls.Add(1)
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
		newStreaming: func(options fetcherOptions, channels fetcherChannels, failoverSignal chan failEvent) togglerFetcher {
			return streamingFetcher
		},
		newPolling: func(options fetcherOptions, channels fetcherChannels) togglerFetcher {
			return pollingFetcher
		},
	}
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool, onTimeout string) {
	t.Helper()
	deadline := time.After(timeout)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		if cond() {
			return
		}
		select {
		case <-deadline:
			t.Fatalf(onTimeout)
		case <-ticker.C:
		}
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
	if streaming.startCalls.Load() != 1 {
		t.Fatalf("expected streaming fetcher to start once, got %d", streaming.startCalls.Load())
	}

	af.cutover()

	if streaming.stopCalls.Load() != 1 {
		t.Fatalf("expected streaming fetcher to stop once, got %d", streaming.stopCalls.Load())
	}

	if polling.startCalls.Load() != 1 {
		t.Fatalf("expected polling fetcher to start once, got %d", polling.startCalls.Load())
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
	if streaming.startCalls.Load() != 1 {
		t.Fatalf("expected streaming fetcher to start once, got %d", streaming.startCalls.Load())
	}

	af.start()
	if streaming.startCalls.Load() != 1 {
		t.Fatalf("expected streaming fetcher to not start again, got %d", streaming.startCalls.Load())
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
	if streaming.stopCalls.Load() != 1 {
		t.Fatalf("expected streaming fetcher to stop once, got %d", streaming.stopCalls.Load())
	}

	af.stop()
	if streaming.stopCalls.Load() != 1 {
		t.Fatalf("expected streaming fetcher to not stop again, got %d", streaming.stopCalls.Load())
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

func TestAdaptiveFetcher_FailoverSignalCutsOverAndWarns(t *testing.T) {
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

	failEvent := &failoverRequest{
		baseFailEvent: baseFailEvent{
			occurredAt: time.Now(),
			message:    "streaming borked",
		},
	}

	select {
	case af.failoverSignalChannel <- failEvent:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout sending failover signal")
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		return polling.startCalls.Load() == 1 && streaming.stopCalls.Load() == 1
	}, "expected cutover to polling after failover signal")

	select {
	case err := <-channels.warnings:
		if err == nil || err.Error() == "" {
			t.Fatalf("expected warning error on failover")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected warning to be emitted on failover")
	}
}
