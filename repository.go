package unleash

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/seatgeek/unleash-client-go/v3/internal/api"
)

type Repository interface {
	GetToggle(key string) *api.Feature
	GetConfig(key string) string
	Close() error
}

type httpRepository struct {
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
}

func NewHttpRepository(options repositoryOptions, channels repositoryChannels) Repository {
	repo := &httpRepository{
		options:            options,
		repositoryChannels: channels,
		close:              make(chan struct{}),
		closed:             make(chan struct{}),
		refreshTicker:      time.NewTicker(options.refreshInterval),
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

func (r *httpRepository) fetchAndReportError() {
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

func (r *httpRepository) sync() {
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
			r.fetchAndReportError()
		}
	}
}

func (r *httpRepository) fetch() error {
	u, _ := r.options.url.Parse("./client/features")

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
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
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil
	}
	if err := statusIsOK(resp); err != nil {
		return err
	}

	var featureResp api.FeatureResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&featureResp); err != nil {
		return err
	}

	r.Lock()
	r.etag = resp.Header.Get("Etag")
	r.options.storage.Reset(featureResp.FeatureMap(), true)
	r.Unlock()
	return nil
}

func statusIsOK(resp *http.Response) error {
	s := resp.StatusCode
	if 200 <= s && s < 300 {
		return nil
	}

	return fmt.Errorf("%s %s returned status code %d", resp.Request.Method, resp.Request.URL, s)
}

func (r *httpRepository) GetToggle(key string) *api.Feature {
	r.RLock()
	defer r.RUnlock()

	if toggle, found := r.options.storage.Get(key); found {
		if feature, ok := toggle.(api.Feature); ok {
			return &feature
		}
	}
	return nil
}

func (r * httpRepository) GetConfig(key string) string {
	r.err(errors.New("Configuration params not implemented for http repositories"))
	return "{}"
}

func (r *httpRepository) Close() error {
	close(r.close)
	r.cancel()
	<-r.closed
	r.refreshTicker.Stop()
	return nil
}
