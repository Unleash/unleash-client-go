package unleash

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// This exists purely because atomics must contain a single concrete type
// we want to be able to both switch fetcher types and avoid mutexes so this
// leaves us with this sad little container struct. Don't "fix" this
type fetchContainer struct {
	f togglerFetcher
}

type fetcherFactory struct {
	newStreaming func(fetcherOptions, fetcherChannels, chan failEvent) togglerFetcher
	newPolling   func(fetcherOptions, fetcherChannels) togglerFetcher
}

func defaultFetcherFactory() fetcherFactory {
	return fetcherFactory{
		newStreaming: func(options fetcherOptions, channels fetcherChannels, failoverChannel chan failEvent) togglerFetcher {
			return newStreamingFetcher(options, channels, failoverChannel)
		},
		newPolling: func(options fetcherOptions, channels fetcherChannels) togglerFetcher {
			return newPollingFetcher(options, channels)
		},
	}
}

type adaptiveFetcher struct {
	currentFetcher        atomic.Pointer[fetchContainer]
	cutoverFetcher        togglerFetcher
	baseOptions           fetcherOptions
	baseChannels          fetcherChannels
	internalReady         chan bool
	internalUpdate        chan bool
	failoverSignalChannel chan failEvent
	ready                 atomic.Bool
	started               bool
	stopped               bool
	ctx                   context.Context
	cancel                context.CancelFunc
	mu                    sync.Mutex
	factory               fetcherFactory
}

func newAdaptiveFetcher(options fetcherOptions, channels fetcherChannels) *adaptiveFetcher {
	return newAdaptiveFetcherWithFactory(options, channels, defaultFetcherFactory())
}

// If you're considering using this in production code, probably don't. This is exposed
// internally so that test code has a seam to inject fake fetchers and control the lifecycle.
// Prefer using newAdaptiveFetcher, which will construct appropriate fetchers for
// real world usage
func newAdaptiveFetcherWithFactory(
	options fetcherOptions,
	channels fetcherChannels,
	factory fetcherFactory,
) *adaptiveFetcher {
	ir := make(chan bool, 1)
	iu := make(chan bool, 1)
	fc := make(chan failEvent, 1)

	ctx, cancel := context.WithCancel(context.Background())

	af := &adaptiveFetcher{
		baseOptions:           options,
		baseChannels:          channels,
		internalReady:         ir,
		internalUpdate:        iu,
		ctx:                   ctx,
		cancel:                cancel,
		factory:               factory,
		failoverSignalChannel: fc,
	}

	initial := af.factory.newStreaming(options, fetcherChannels{
		ready:         ir,
		update:        iu,
		errorChannels: channels.errorChannels,
	}, af.failoverSignalChannel)

	af.currentFetcher.Store(&fetchContainer{f: initial})

	return af
}

func (af *adaptiveFetcher) superviseFetchers() {
	for {
		select {
		case <-af.internalReady:
			af.handleReadyEvent()
		case <-af.internalUpdate:
			af.baseChannels.update <- true
		case errorEvent := <-af.failoverSignalChannel:
			af.baseChannels.warnings <- fmt.Errorf("Adaptive fetcher failover triggered: %s", errorEvent.Message())
			af.cutover()
		case <-af.ctx.Done():
			return
		}
	}
}

func (af *adaptiveFetcher) handleReadyEvent() {
	af.mu.Lock()
	defer af.mu.Unlock()

	// catch the first ready event - this necessarily must be the startup fetcher
	// telling us that it's hydrated. From a user perspective this *is* the ready event
	// after this point we're only ever waiting for a cutover fetcher to signal that
	// it's ready to take over, which the user doesn't need to know about
	if !af.ready.Load() {
		af.ready.Store(true)
		// Guard with hasHydrated(): streaming's ready and a failover signal can land
		// simultaneously; if failover is processed first cutoverFetcher is set but polling
		// hasn't hydrated — the signal came from streaming, so switching to polling's empty
		// cache would be wrong. Only switch when polling is confirmed to have hydrated.
		if af.cutoverFetcher != nil && af.cutoverFetcher.hasHydrated() {
			af.currentFetcher.Load().f.stop()
			af.currentFetcher.Store(&fetchContainer{f: af.cutoverFetcher})
			af.cutoverFetcher = nil
		}
		af.baseChannels.ready <- true
		return
	}

	if af.cutoverFetcher != nil {
		af.currentFetcher.Load().f.stop()
		af.currentFetcher.Store(&fetchContainer{f: af.cutoverFetcher})
		af.cutoverFetcher = nil
	}
}

func (af *adaptiveFetcher) cutover() {
	af.mu.Lock()
	defer af.mu.Unlock()
	if af.stopped {
		return
	}

	current := af.currentFetcher.Load().f

	if _, ok := current.(*pollingFetcher); ok {
		return
	}

	if af.cutoverFetcher != nil {
		return
	}

	// User explicitly disabled polling: honour that even during streaming failover.
	if af.baseOptions.disablePolling {
		if !af.ready.Load() {
			af.ready.Store(true)
			af.baseChannels.ready <- true
		}
		return
	}

	// Failover must never block: with synchronousFetch set, start() fetches inline then
	// sends to af.internalReady. If that channel is already full (streaming's buffered
	// ready hasn't been drained), start() blocks — superviseFetchers is the only reader
	// but it's stuck inside cutover() holding af.mu, so Close() deadlocks too.
	failoverOptions := af.baseOptions
	failoverOptions.synchronousFetch = false
	next := af.factory.newPolling(failoverOptions, fetcherChannels{
		ready:         af.internalReady,
		update:        af.internalUpdate,
		errorChannels: af.baseChannels.errorChannels,
	})

	af.cutoverFetcher = next

	next.start()
}

func (af *adaptiveFetcher) start() {
	af.mu.Lock()
	defer af.mu.Unlock()
	if af.started || af.stopped {
		return
	}
	af.started = true

	go af.superviseFetchers()
	af.currentFetcher.Load().f.start()
}

func (af *adaptiveFetcher) hasHydrated() bool {
	return af.currentFetcher.Load().f.hasHydrated()
}

func (af *adaptiveFetcher) snapshot() *FeatureMemoryState {
	return af.currentFetcher.Load().f.snapshot()
}

func (af *adaptiveFetcher) stop() {
	af.mu.Lock()
	defer af.mu.Unlock()

	if af.stopped {
		return
	}
	af.stopped = true
	af.cancel()

	if af.cutoverFetcher != nil {
		af.cutoverFetcher.stop()
		af.cutoverFetcher = nil
	}

	af.currentFetcher.Load().f.stop()
}
