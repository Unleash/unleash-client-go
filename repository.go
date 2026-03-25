package unleash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Unleash/unleash-go-sdk/v6/api"
)

var SEGMENT_CLIENT_SPEC_VERSION = "4.3.1"

var (
	errNoChange = errors.New("no change")
)

type togglerFetcher interface {
	sync()
	list() []api.Feature
	snapshot() *FeatureMemoryState
	Close() error
}

type repository struct {
	repositoryChannels
	sync.RWMutex
	options         repositoryOptions
	etag            string
	close           chan struct{}
	closed          chan struct{}
	ctx             context.Context
	cancel          func()
	isReady         bool
	refreshTicker   *time.Ticker
	errors          float64
	maxSkips        float64
	skips           float64
	streamingClient *streamingClient
	isStreaming     bool
	deltaProcessor  *deltaProcessor
	featureState    atomic.Value // this should always hold an instance of *FeatureMemoryState
}

func newRepository(options repositoryOptions, channels repositoryChannels) *repository {
	repo := &repository{
		options:            options,
		repositoryChannels: channels,
		close:              make(chan struct{}),
		closed:             make(chan struct{}),
		refreshTicker:      time.NewTicker(options.refreshInterval),
		errors:             0,
		maxSkips:           10,
		skips:              0,
		isStreaming:        options.isStreaming,
	}
	ctx, cancel := context.WithCancel(context.Background())
	repo.ctx = ctx
	repo.cancel = cancel

	if options.httpClient == nil {
		repo.options.httpClient = http.DefaultClient
	}

	if options.storage == nil {
		repo.options.storage = &DefaultStorage{}
	}

	repo.options.storage.Init(options.backupPath, options.appName)
	repo.deltaProcessor = newDeltaProcessor(repo, channels)

	if loadedState, err := repo.options.storage.Load(); err == nil && loadedState != nil {
		repo.updateState(loadedState)
	} else {
		repo.featureState.Store(&FeatureMemoryState{
			Features: make(map[string]*api.Feature),
			Segments: make(map[int][]api.Constraint),
		})
	}

	// Delta processor needs to be collapsed into this module, it's far too jealous of this domain at the moment
	if repo.isStreaming {
		repo.streamingClient = newStreamingClient(
			options,
			channels,
			repo.deltaProcessor)
	}

	go repo.sync()

	return repo
}

func (r *repository) updateState(features *api.FeatureResponse) {
	state := &FeatureMemoryState{
		Features: features.FeatureMap(),
		Segments: features.SegmentsMap(),
	}

	r.featureState.Store(state)
}

func (r *repository) saveState(features *api.FeatureResponse) error {
	r.updateState(features)
	return r.options.storage.Persist(features)
}

func (r *repository) fetchAndReportError() {
	var (
		isUnchanged bool
		err         error
	)

	err = r.fetch()

	// Extract unchanged error from error
	if err != nil {
		isUnchanged = errors.Is(err, errNoChange)
		if isUnchanged {
			err = nil
		}
	}

	if err != nil {
		if urlErr, ok := err.(*url.Error); !(ok && urlErr.Err == context.Canceled) {
			r.err(err)
		}
	} else if !r.isReady {
		r.isReady = true
		r.ready <- true
	} else if !isUnchanged {
		r.update <- true
	}
}

func (r *repository) sync() {
	// Single read lock to determine initial mode
	r.RLock()
	isStreaming := r.isStreaming
	streamingClient := r.streamingClient
	r.RUnlock()

	// Start streaming mode if enabled
	// The eventsource library handles all reconnections automatically with backoff and jitter
	if isStreaming && streamingClient != nil {
		if err := streamingClient.start(r.options.storage); err != nil {
			r.err(fmt.Errorf("failed to start streaming client: %w", err))
		}
	}

	// Use polling if not streaming
	if !isStreaming {
		r.fetchAndReportError()
	}

	for {
		select {
		case <-r.close:
			if r.streamingClient != nil {
				r.streamingClient.stop()
			}
			close(r.closed)
			return
		case <-r.refreshTicker.C:
			// Only poll if not in streaming mode
			r.RLock()
			shouldPoll := !r.isStreaming
			r.RUnlock()

			if shouldPoll {
				if r.skips == 0 {
					r.fetchAndReportError()
				} else {
					r.decrementSkips()
				}
			}
		}
	}
}

func (r *repository) backoff() {
	r.errors = math.Min(r.maxSkips, r.errors+1)
	r.skips = r.errors
}

func (r *repository) successfulFetch() {
	r.errors = math.Max(0, r.errors-1)
	r.skips = r.errors
}

func (r *repository) decrementSkips() {
	r.skips = math.Max(0, r.skips-1)
}
func (r *repository) configurationError() {
	r.errors = r.maxSkips
	r.skips = r.errors
}

func (r *repository) fetch() error {
	u, _ := r.options.url.Parse(getFetchURLPath(r.options.projectName))

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}
	req = req.WithContext(r.ctx)

	req.Header.Add("UNLEASH-APPNAME", r.options.appName)
	req.Header.Add("UNLEASH-INSTANCEID", r.options.instanceId)
	req.Header.Add("Unleash-Interval", fmt.Sprintf("%d", r.options.refreshInterval.Milliseconds()))
	req.Header.Add("User-Agent", r.options.appName)
	// Needs to reference a version of the client specifications that include
	// global segments
	req.Header.Add("Unleash-Client-Spec", SEGMENT_CLIENT_SPEC_VERSION)

	for k, v := range r.options.headers {
		req.Header[k] = v
	}

	if r.etag != "" {
		req.Header.Add("If-None-Match", r.etag)
	}

	resp, err := r.options.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return errNoChange
	}
	if err := r.statusIsOK(resp); err != nil {
		return err
	}

	var featureResp api.FeatureResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&featureResp); err != nil {
		return err
	}

	r.Lock()
	r.etag = resp.Header.Get("Etag")
	r.saveState(&featureResp)
	r.successfulFetch()
	r.Unlock()
	return nil
}

func (r *repository) statusIsOK(resp *http.Response) error {
	s := resp.StatusCode
	if http.StatusOK <= s && s < http.StatusMultipleChoices {
		return nil
	} else if s == http.StatusUnauthorized || s == http.StatusForbidden || s == http.StatusNotFound {
		r.configurationError()
		return fmt.Errorf("%s %s returned status code %d your SDK is most likely misconfigured, backing off to maximum (%f times our interval)", resp.Request.Method, resp.Request.URL, s, r.maxSkips)
	} else if s == http.StatusTooManyRequests || s >= http.StatusInternalServerError {
		r.backoff()
		return fmt.Errorf("%s %s returned status code %d, backing off (%f times our interval)", resp.Request.Method, resp.Request.URL, s, r.errors)
	}

	return fmt.Errorf("%s %s returned status code %d", resp.Request.Method, resp.Request.URL, s)
}

func (r *repository) list() []api.Feature {

	snapshot := r.snapshot()
	raw := snapshot.Features
	features := make([]api.Feature, 0, len(raw))

	// we're doing an explicit copy here, this function should not be on a hot path
	// and we want to avoid exposing internal pointers or changing too much of the public API
	for _, feature := range raw {
		if feature == nil {
			continue
		}
		features = append(features, *feature)
	}
	return features
}

func (r *repository) snapshot() *FeatureMemoryState {
	v := r.featureState.Load()
	if v == nil {
		empty := &FeatureMemoryState{
			Features: make(map[string]*api.Feature),
			Segments: make(map[int][]api.Constraint),
		}
		r.featureState.Store(empty)
		return empty
	}
	return v.(*FeatureMemoryState)
}

func (r *repository) Close() error {
	close(r.close)
	r.cancel()
	<-r.closed
	r.refreshTicker.Stop()
	return nil
}
