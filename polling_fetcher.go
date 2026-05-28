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
	options          fetcherOptions
	etag             string
	close            chan struct{}
	closed           chan struct{}
	ctx              context.Context
	cancel           func()
	isReady          bool
	refreshTicker    *time.Ticker
	errors           float64
	maxSkips         float64
	skips            float64
	featureCache     *featureCache
	disablePolling   bool
	synchronousFetch bool
}

func newPollingFetcher(options fetcherOptions, channels fetcherChannels) *pollingFetcher {
	f := &pollingFetcher{
		options:          options,
		fetcherChannels:  channels,
		close:            make(chan struct{}),
		closed:           make(chan struct{}),
		refreshTicker:    time.NewTicker(options.refreshInterval),
		errors:           0,
		maxSkips:         10,
		skips:            0,
		disablePolling:   options.disablePolling,
		synchronousFetch: options.synchronousFetch,
	}
	ctx, cancel := context.WithCancel(context.Background())
	f.ctx = ctx
	f.cancel = cancel

	if options.httpClient == nil {
		f.options.httpClient = http.DefaultClient
	}

	var apiResponse *api.FeatureResponse

	if loadedState, err := options.storage.Load(); err == nil && loadedState != nil {
		apiResponse = loadedState
	} else {
		apiResponse = &api.FeatureResponse{
			Features: []api.Feature{},
			Segments: []api.Segment{},
		}
	}

	f.featureCache = newFeatureCache(apiResponse)

	return f
}

func (r *pollingFetcher) saveState(features *api.FeatureResponse) error {
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
	// Initial fetch is skipped here when synchronousFetch already did it in start().
	if !r.synchronousFetch {
		r.fetchAndReportError()
	}
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
	if r.synchronousFetch {
		r.fetchAndReportError()
	}
	if r.disablePolling {
		r.refreshTicker.Stop() // no polling loop will ever read this ticker
		// Signal ready with backup state when the synchronous fetch didn't succeed.
		if !r.isReady {
			r.isReady = true
			r.ready <- true
		}
		close(r.closed)
		return
	}
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

	var featureResp api.ApiResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&featureResp); err != nil {
		return false, err
	}

	r.Lock()
	r.etag = resp.Header.Get("Etag")

	apiResponse, err := r.featureCache.updateFromApiResponse(&featureResp)
	if err != nil {
		r.Unlock()
		return false, err
	}
	r.saveState(apiResponse)

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
	return r.featureCache.snapshot()
}

func (r *pollingFetcher) stop() {
	close(r.close)
	r.cancel()
	<-r.closed
	r.refreshTicker.Stop()
}
