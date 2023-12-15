package unleash

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Unleash/unleash-client-go/v4/api"
)

var SEGMENT_CLIENT_SPEC_VERSION = "4.3.1"

type repository struct {
	repositoryChannels
	sync.RWMutex
	options       repositoryOptions
	etag          string
	close         chan struct{}
	closed        chan struct{}
	ctx           context.Context
	cancel        func()
	isReady       bool
	refreshTicker *time.Ticker
	segments      map[int][]api.Constraint
	errors        float64
	maxSkips      float64
	skips         float64
}

func newRepository(options repositoryOptions, channels repositoryChannels) *repository {
	repo := &repository{
		options:            options,
		repositoryChannels: channels,
		close:              make(chan struct{}),
		closed:             make(chan struct{}),
		refreshTicker:      time.NewTicker(options.refreshInterval),
		segments:           map[int][]api.Constraint{},
		errors:             0,
		maxSkips:           10,
		skips:              0,
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

	go repo.sync()

	return repo
}

func (r *repository) fetchAndReportError() {
	err := r.fetch()
	if err != nil {
		if urlErr, ok := err.(*url.Error); !(ok && urlErr.Err == context.Canceled) {
			r.err(err)
		}
	}
	if !r.isReady && err == nil {
		r.isReady = true
		r.ready <- true
	}
}

func (r *repository) sync() {
	r.fetchAndReportError()
	for {
		select {
		case <-r.close:
			if err := r.options.storage.Persist(); err != nil {
				r.err(err)
			}
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
	req.Header.Add("User-Agent", r.options.appName)
	// Needs to reference a version of the client specifications that include
	// global segments
	req.Header.Add("Unleash-Client-Spec", SEGMENT_CLIENT_SPEC_VERSION)

	for k, v := range r.options.customHeaders {
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
		return nil
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
	r.segments = featureResp.SegmentsMap()
	r.options.storage.Reset(featureResp.FeatureMap(), true)
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

func (r *repository) getToggle(key string) *api.Feature {
	r.RLock()
	defer r.RUnlock()

	if toggle, found := r.options.storage.Get(key); found {
		if feature, ok := toggle.(api.Feature); ok {
			return &feature
		}
	}
	return nil
}

func (r *repository) resolveSegmentConstraints(strategy api.Strategy) ([]api.Constraint, error) {
	segmentConstraints := []api.Constraint{}

	for _, segmentId := range strategy.Segments {
		if resolvedConstraints, ok := r.segments[segmentId]; ok {
			segmentConstraints = append(segmentConstraints, resolvedConstraints...)
		} else {
			return segmentConstraints, fmt.Errorf("segment does not exist")
		}
	}

	return segmentConstraints, nil
}

func (r *repository) list() []api.Feature {
	r.RLock()
	defer r.RUnlock()

	var features []api.Feature
	for _, feature := range r.options.storage.List() {
		features = append(features, feature.(api.Feature))
	}
	return features
}

func (r *repository) Close() error {
	close(r.close)
	r.cancel()
	<-r.closed
	r.refreshTicker.Stop()
	return nil
}
