package unleash

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Unleash/unleash-client-go/internal/api"
)

type repository struct {
	repositoryChannels
	sync.RWMutex
	options repositoryOptions
	etag    string
	close   chan bool
}

func newRepository(options repositoryOptions, channels repositoryChannels) *repository {
	fmt.Println("initializing a new repository...")
	repo := &repository{
		options:            options,
		repositoryChannels: channels,
	}

	if options.httpClient == nil {
		repo.options.httpClient = http.DefaultClient
	}

	if options.storage == nil {
		repo.options.storage = &defaultStorage{}
	}

	repo.options.storage.Init(options.backupPath, options.appName)
	repo.close = make(chan bool)
	go repo.sync()

	return repo
}

func (r *repository) sync() {
	defer r.cleanup()

	refreshTimer := time.NewTimer(r.options.refreshInterval)
	fmt.Printf("async(will refresh repository every %v)\n", r.options.refreshInterval)

	r.fetch()
	r.ready <- true

	for {
		select {
		case <-r.close:
			refreshTimer.Stop()
			fmt.Println("async(stopped the repository sync routine...)")
			return
		case <-refreshTimer.C:
			r.fetch()
		}
	}
}

func (r *repository) cleanup() {
	if err := r.options.storage.Persist(); err != nil {
		r.err(err)
	}
}

func (r *repository) fetch() {
	fmt.Printf("fetching features...\n")
	u, _ := r.options.url.Parse("./features")

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		r.err(err)
		return
	}

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
	r.close <- true
	fmt.Println("repository closed")
	return nil
}
