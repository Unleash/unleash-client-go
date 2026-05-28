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
	hydrated   atomic.Bool
	state      *FeatureMemoryState
}

func (f *fakeFetcher) start() {
	f.startCalls.Add(1)
}

func (f *fakeFetcher) stop() {
	f.stopCalls.Add(1)
}

func (f *fakeFetcher) hasHydrated() bool {
	return f.hydrated.Load()
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

func waitForReady(t *testing.T, ch chan bool, label string) {
	t.Helper()
	select {
	case ch <- true:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout sending %s signal", label)
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

	waitForReady(t, af.internalReady, "initial ready")

	af.cutover()

	if polling.startCalls.Load() != 1 {
		t.Fatalf("expected polling fetcher to start once, got %d", polling.startCalls.Load())
	}

	if streaming.stopCalls.Load() != 0 {
		t.Fatalf("expected streaming fetcher to keep running until polling is ready, got %d", streaming.stopCalls.Load())
	}

	waitForReady(t, af.internalReady, "polling ready")

	waitFor(t, 500*time.Millisecond, func() bool {
		return streaming.stopCalls.Load() == 1
	}, "expected streaming fetcher to stop after polling ready")

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
	waitForReady(t, af.internalReady, "initial ready")
	af.cutover()

	got := af.snapshot()
	if _, ok := got.Features["streaming"]; !ok {
		t.Fatalf("expected snapshot from streaming fetcher before polling ready")
	}

	waitForReady(t, af.internalReady, "polling ready")

	waitFor(t, 500*time.Millisecond, func() bool {
		return polling.startCalls.Load() == 1 && streaming.stopCalls.Load() == 1
	}, "expected cutover to polling after polling ready")

	got = af.snapshot()
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

	waitForReady(t, af.internalReady, "initial ready")

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
		return polling.startCalls.Load() == 1
	}, "expected polling fetcher to start after failover signal")

	if streaming.stopCalls.Load() != 0 {
		t.Fatalf("expected streaming fetcher to keep running until polling is ready, got %d", streaming.stopCalls.Load())
	}

	waitForReady(t, af.internalReady, "polling ready")

	waitFor(t, 500*time.Millisecond, func() bool {
		return streaming.stopCalls.Load() == 1
	}, "expected streaming fetcher to stop after polling ready")

	select {
	case err := <-channels.warnings:
		if err == nil || err.Error() == "" {
			t.Fatalf("expected warning error on failover")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected warning to be emitted on failover")
	}
}

func TestAdaptiveFetcher_CutoverWaitsForPollingReady(t *testing.T) {
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

	waitForReady(t, af.internalReady, "initial ready")

	af.cutover()

	if streaming.stopCalls.Load() != 0 {
		t.Fatalf("expected streaming fetcher to continue until polling ready, got %d", streaming.stopCalls.Load())
	}

	got := af.snapshot()
	if _, ok := got.Features["streaming"]; !ok {
		t.Fatalf("expected snapshot from streaming fetcher before polling ready")
	}
}

// TestAdaptiveFetcher_FailoverBeforeHydrationSwitchesToPollingOnReady tests scenario B:
// streaming fails before ever sending a ready event; the cutover polling fetcher
// hydrates and its ready event must fire user-facing ready with the polling snapshot.
func TestAdaptiveFetcher_FailoverBeforeHydrationSwitchesToPollingOnReady(t *testing.T) {
	streaming := &fakeFetcher{
		state: &FeatureMemoryState{
			Features: map[string]*api.Feature{},
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

	// Send failover BEFORE streaming fires any ready signal.
	failoverEvent := &failoverRequest{
		baseFailEvent: baseFailEvent{
			occurredAt: time.Now(),
			message:    "streaming failed before hydration",
		},
	}
	select {
	case af.failoverSignalChannel <- failoverEvent:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timeout sending failover signal")
	}

	// Polling should start.
	waitFor(t, 500*time.Millisecond, func() bool {
		return polling.startCalls.Load() == 1
	}, "expected polling fetcher to start after failover signal")

	// User-facing ready must not have fired yet (polling hasn't hydrated).
	select {
	case <-channels.ready:
		t.Fatalf("ready should not fire until polling hydrates")
	default:
	}

	// Simulate polling hydrating and firing its first ready signal.
	polling.hydrated.Store(true)
	waitForReady(t, af.internalReady, "polling first ready")

	// User-facing ready must now fire.
	select {
	case <-channels.ready:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected user-facing ready after polling hydration")
	}

	// Streaming must have been stopped (it was replaced by polling).
	waitFor(t, 500*time.Millisecond, func() bool {
		return streaming.stopCalls.Load() == 1
	}, "expected streaming fetcher to stop after polling took over")

	// Snapshot must reflect polling data.
	got := af.snapshot()
	if _, ok := got.Features["polling"]; !ok {
		t.Fatalf("expected polling snapshot after failover-before-hydration cutover, got %v", got.Features)
	}
}
