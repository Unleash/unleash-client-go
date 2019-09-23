package unleash

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/Unleash/unleash-client-go/v3/internal/api"
)

type repository struct {
	repositoryChannels
	sync.RWMutex
	options repositoryOptions
	etag    string
	close   chan struct{}
	closed  chan struct{}
	ctx     context.Context
	cancel  func()
}

func newRepository(options repositoryOptions, channels repositoryChannels) *repository {
	repo := &repository{
		options:            options,
		repositoryChannels: channels,
		close:              make(chan struct{}),
		closed:             make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	repo.ctx = ctx
	repo.cancel = cancel

	if options.httpClient == nil {
		repo.options.httpClient = http.DefaultClient
	}

	if options.storage == nil {
		repo.options.storage = &defaultStorage{}
	}

	repo.options.storage.Init(options.backupPath, options.appName)

	go repo.sync()

	return repo
}

func (r *repository) sync() {
	r.fetch()
	r.ready <- true

	for {
		refreshTimer := time.NewTimer(r.options.refreshInterval)

		select {
		case <-r.close:
			if err := r.options.storage.Persist(); err != nil {
				r.err(err)
			}
			close(r.closed)
			return
		case <-refreshTimer.C:
			r.fetch()
		}
	}
}

func (r *repository) fetch() {
	u, _ := r.options.url.Parse("./client/features")

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		r.err(err)
		return
	}
	req = req.WithContext(r.ctx)

	req.Header.Add("UNLEASH-APPNAME", r.options.appName)
	req.Header.Add("UNLEASH-INSTANCEID", r.options.instanceId)
	req.Header.Add("User-Agent", r.options.appName)

	for k, v := range r.options.customHeaders {
		req.Header[k] = v
	}

	if r.etag != "" {
		req.Header.Add("If-None-Match", r.etag)
	}

	resp, err := r.options.httpClient.Do(req)
	if err != nil {
		r.err(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return
	}

	var featureResp api.FeatureResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&featureResp); err != nil {
		r.err(err)
		return
	}

	r.Lock()
	r.etag = resp.Header.Get("Etag")
	r.options.storage.Reset(featureResp.FeatureMap(), true)
	r.Unlock()
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

func (r *repository) Close() error {
	close(r.close)
	r.cancel()
	<-r.closed
	return nil
}
