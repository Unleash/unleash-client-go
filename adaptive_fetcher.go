package unleash

import (
	"context"
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
	newStreaming func(fetcherOptions, fetcherChannels, chan streamError) togglerFetcher
	newPolling   func(fetcherOptions, fetcherChannels) togglerFetcher
}

func defaultFetcherFactory() fetcherFactory {
	return fetcherFactory{
		newStreaming: func(options fetcherOptions, channels fetcherChannels, streamErrorChannel chan streamError) togglerFetcher {
			return newStreamingFetcher(options, channels, streamErrorChannel)
		},
		newPolling: func(options fetcherOptions, channels fetcherChannels) togglerFetcher {
			return newPollingFetcher(options, channels)
		},
	}
}

type adaptiveFetcher struct {
	currentFetcher        atomic.Pointer[fetchContainer]
	baseOptions           fetcherOptions
	baseChannels          fetcherChannels
	internalReady         chan bool
	internalUpdate        chan bool
	failoverSignalChannel chan streamError
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
	se := make(chan streamError, 1)

	ctx, cancel := context.WithCancel(context.Background())

	af := &adaptiveFetcher{
		baseOptions:           options,
		baseChannels:          channels,
		internalReady:         ir,
		internalUpdate:        iu,
		ctx:                   ctx,
		cancel:                cancel,
		factory:               factory,
		failoverSignalChannel: se,
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
			if af.ready.CompareAndSwap(false, true) {
				af.baseChannels.ready <- true
			}
		case <-af.internalUpdate:
			af.baseChannels.update <- true
		case <-af.failoverSignalChannel:
			// do nothing for now but this is where cutover will get triggered from
		case <-af.ctx.Done():
			return
		}
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

	next := af.factory.newPolling(af.baseOptions, fetcherChannels{
		ready:         af.internalReady,
		update:        af.internalUpdate,
		errorChannels: af.baseChannels.errorChannels,
	})

	af.currentFetcher.Store(&fetchContainer{f: next})

	next.start()
	current.stop()
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
	af.currentFetcher.Load().f.stop()
}
