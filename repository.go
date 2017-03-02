package unleash_client_go

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/unleash/unleash-client-go/internal/api"
	"sync"
)

type RepositoryOptions struct {
	BackupPath      string
	Url             url.URL
	AppName         string
	InstanceId      string
	RefreshInterval time.Duration
	Storage         Storage
	HttpClient      *http.Client
}

type repository struct {
	errorEmitterImpl
	sync.RWMutex
	options RepositoryOptions
	etag    string
	ready   chan bool
	close   chan bool
}

func NewRepository(options RepositoryOptions) *repository {
	repo := &repository{
		errorEmitterImpl: *newErrorEmitter(),
		options:          options,
		ready:            make(chan bool, 1),
	}

	if options.HttpClient == nil {
		repo.options.HttpClient = http.DefaultClient
	}

	if options.Storage == nil {
		repo.options.Storage = &defaultStorage{}
	}

	repo.options.Storage.Init(options.BackupPath, options.AppName)

	go repo.sync()

	return repo
}

func (r *repository) sync() {
	defer r.cleanup()

	r.fetch()
	r.ready <- true

	for {
		refreshTimer := time.NewTimer(r.options.RefreshInterval)

		select {
		case <-r.close:
			return
		case <-refreshTimer.C:
			r.fetch()
		}
	}
}

func (r *repository) cleanup() {
	if err := r.options.Storage.Persist(); err != nil {
		r.err(err)
	}
}

func (r *repository) fetch() {
	u, _ := r.options.Url.Parse("./features")

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		r.err(err)
		return
	}

	req.Header.Add("UNLEASH-APPNAME", r.options.AppName)
	req.Header.Add("UNLEASH-INSTANCEID", r.options.InstanceId)

	if r.etag != "" {
		req.Header.Add("If-None-Match", r.etag)
	}

	resp, err := r.options.HttpClient.Do(req)
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
	r.options.Storage.Reset(featureResp.FeatureMap(), true)
	r.Unlock()
}

func (r repository) GetToggle(key string) *api.Feature {
	r.RLock()
	defer r.RUnlock()

	if toggle, found := r.options.Storage.Get(key); found {
		if feature, ok := toggle.(api.Feature); ok {
			return &feature
		}
	}
	return nil
}

func (r repository) Close() error {
	r.close <- true
	return nil
}
