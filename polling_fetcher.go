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
	start()
	snapshot() *FeatureMemoryState
	stop()
}

type pollingFetcher struct {
	fetcherChannels
	sync.RWMutex
	options       fetcherOptions
	etag          string
	close         chan struct{}
	closed        chan struct{}
	ctx           context.Context
	cancel        func()
	isReady       bool
	refreshTicker *time.Ticker
	errors        float64
	maxSkips      float64
	skips         float64
	featureState  atomic.Value // this should always hold an instance of *FeatureMemoryState
}

func newPollingFetcher(options fetcherOptions, channels fetcherChannels) *pollingFetcher {
	f := &pollingFetcher{
		options:         options,
		fetcherChannels: channels,
		close:           make(chan struct{}),
		closed:          make(chan struct{}),
		refreshTicker:   time.NewTicker(options.refreshInterval),
		errors:          0,
		maxSkips:        10,
		skips:           0,
	}
	ctx, cancel := context.WithCancel(context.Background())
	f.ctx = ctx
	f.cancel = cancel

	if options.httpClient == nil {
		f.options.httpClient = http.DefaultClient
	}

	if loadedState, err := f.options.storage.Load(); err == nil && loadedState != nil {
		f.updateState(loadedState)
	} else {
		f.featureState.Store(&FeatureMemoryState{
			Features: make(map[string]*api.Feature),
			Segments: make(map[int][]api.Constraint),
		})
	}

	return f
}

func (r *pollingFetcher) updateState(features *api.FeatureResponse) {
	state := &FeatureMemoryState{
		Features: features.FeatureMap(),
		Segments: features.SegmentsMap(),
	}

	r.featureState.Store(state)
}

func (r *pollingFetcher) saveState(features *api.FeatureResponse) error {
	r.updateState(features)
	return r.options.storage.Persist(features)
}

func (r *pollingFetcher) fetchAndReportError() {
	changed, err := r.fetch()

	if err != nil {
		if urlErr, ok := err.(*url.Error); !(ok && urlErr.Err == context.Canceled) {
			r.err(err)
		}
	} else if !r.isReady {
		r.isReady = true
		r.ready <- true
	} else if changed {
		r.update <- true
	}
}

func (r *pollingFetcher) runPollingLoop() {
	// Initial fetch to populate state and signal readiness.
	r.fetchAndReportError()
	for {
		select {
		case <-r.close:
			close(r.closed)
			return
		case <-r.refreshTicker.C:

			if r.skips == 0 {
				r.fetchAndReportError()
			} else {
				r.decrementSkips()
			}
		}
	}
}

func (r *pollingFetcher) start() {
	go r.runPollingLoop()
}

func (r *pollingFetcher) backoff() {
	r.errors = math.Min(r.maxSkips, r.errors+1)
	r.skips = r.errors
}

func (r *pollingFetcher) successfulFetch() {
	r.errors = math.Max(0, r.errors-1)
	r.skips = r.errors
}

func (r *pollingFetcher) decrementSkips() {
	r.skips = math.Max(0, r.skips-1)
}
func (r *pollingFetcher) configurationError() {
	r.errors = r.maxSkips
	r.skips = r.errors
}

func (r *pollingFetcher) fetch() (bool, error) {
	u, _ := r.options.url.Parse(getFetchURLPath(r.options.projectName))

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return false, err
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
		return false, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return false, nil
	}
	if err := r.statusIsOK(resp); err != nil {
		return false, err
	}

	var featureResp api.FeatureResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&featureResp); err != nil {
		return false, err
	}

	r.Lock()
	r.etag = resp.Header.Get("Etag")
	r.saveState(&featureResp)
	r.successfulFetch()
	r.Unlock()
	return true, nil
}

func (r *pollingFetcher) statusIsOK(resp *http.Response) error {
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

func (r *pollingFetcher) snapshot() *FeatureMemoryState {
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

func (r *pollingFetcher) stop() {
	close(r.close)
	r.cancel()
	<-r.closed
	r.refreshTicker.Stop()
}
