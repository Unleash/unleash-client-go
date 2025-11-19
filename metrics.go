package unleash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5/internal/api"
)

// MetricsData represents the data sent to the unleash server.
type MetricsData struct {
	// AppName is the name of the application.
	AppName string `json:"appName"`

	// InstanceID is the instance identifier.
	InstanceID string `json:"instanceId"`

	// ConnectionId is the connection id for instance.
	ConnectionId string `json:"connectionId"`

	// Bucket is the payload data sent to the server.
	Bucket api.Bucket `json:"bucket"`

	// The runtime version of our Platform
	PlatformVersion string `json:"platformVersion"`

	// The runtime name of our Platform
	PlatformName string `json:"platformName"`

	// Which version of Yggdrasil is being used
	YggdrasilVersion *string `json:"yggdrasilVersion"`

	// Optional field that describes the sdk version (name:version)
	SDKVersion string `json:"sdkVersion"`

	// Which version of the Unleash-Client-Spec is this SDK validated against
	SpecVersion string `json:"specVersion"`
}

// ClientData represents the data sent to the unleash during registration.
type ClientData struct {
	// AppName is the name of the application.
	AppName string `json:"appName"`

	// InstanceID is the instance identifier.
	InstanceID string `json:"instanceId"`

	// ConnectionId is the connection id for instance.
	ConnectionId string `json:"connectionId"`

	// Optional field that describes the sdk version (name:version)
	SDKVersion string `json:"sdkVersion"`

	// Strategies is a list of names of the strategies supported by the client.
	Strategies []string `json:"strategies"`

	// Started indicates the time at which the client was created.
	Started time.Time `json:"started"`

	// Interval specifies the time interval (in ms) that the client is using for refreshing
	// feature toggles.
	Interval int64 `json:"interval"`

	PlatformVersion string `json:"platformVersion"`

	PlatformName string `json:"platformName"`

	YggdrasilVersion *string `json:"yggdrasilVersion"`

	// Which version of the Unleash-Client-Spec is this SDK validated against
	SpecVersion string `json:"specVersion"`
}

type metric struct {
	// Name is the name of the feature toggle.
	Name string

	// Enabled indicates whether the feature was enabled or not.
	Enabled bool
}

type toggleCounters struct {
	yes int64
	no  int64

	mu       sync.Mutex
	variants map[string]int64
}

type metrics struct {
	metricsChannels
	options         metricsOptions
	started         time.Time
	last_close_time time.Time
	counters        sync.Map // map[string]*toggleCounters
	ticker          *time.Ticker
	close           chan struct{}
	closed          chan struct{}
	ctx             context.Context
	cancel          func()
	maxSkips        float64
	errors          float64
	skips           float64
}

func newMetrics(options metricsOptions, channels metricsChannels) *metrics {
	m := &metrics{
		metricsChannels: channels,
		options:         options,
		started:         time.Now(),
		close:           make(chan struct{}),
		closed:          make(chan struct{}),
		maxSkips:        10,
		errors:          0,
		skips:           0,
		last_close_time: time.Now(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel

	if m.options.httpClient == nil {
		m.options.httpClient = http.DefaultClient
	}

	if m.options.metricsInterval <= 0 {
		m.options.disableMetrics = true
	}
	if !m.options.disableMetrics {
		m.ticker = time.NewTicker(m.options.metricsInterval)
		m.registerInstance()
		go m.sync()
	}

	return m
}

func (m *metrics) Close() error {
	if !m.options.disableMetrics {
		m.ticker.Stop()
		m.cancel()
		close(m.close)
		<-m.closed
	}
	return nil
}

func (m *metrics) sync() {
	for {
		select {
		case <-m.ticker.C:
			if m.skips == 0 {
				m.sendMetrics()
			} else {
				m.decrementSkip()
			}
		case <-m.close:
			close(m.closed)
			return
		}
	}
}

func (m *metrics) registerInstance() {
	u, _ := m.options.url.Parse("./client/register")
	payload := m.getClientData()
	resp, err := m.doPost(u, payload)

	if err != nil {
		m.err(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusMultipleChoices {
		m.warn(fmt.Errorf("%s return %d", u.String(), resp.StatusCode))
	}

	m.registered <- payload
}
func (m *metrics) backoff() {
	m.errors = math.Min(m.maxSkips, m.errors+1)
	m.skips = m.errors
}

func (m *metrics) configurationError() {
	m.errors = m.maxSkips
	m.skips = m.errors
}

func (m *metrics) successfulPost() {
	m.errors = math.Max(0, m.errors-1)
	m.skips = m.errors
}

func (m *metrics) decrementSkip() {
	m.skips = math.Max(0, m.skips-1)
}

// This does not remove stale toggle names from the map. I don't think there's a safe, lock free way to do that
// The consequence is that if the user archives a lot of toggles this internal representation will not lose those
// toggles until the process is terminated. In practice, I don't believe this is a big problem, just means a
// little bit more memory is held than necessary
func (m *metrics) buildBucketAndReset(last_close_time time.Time) (api.Bucket, bool) {
	bucket := api.Bucket{
		Start:   last_close_time,
		Toggles: make(map[string]api.ToggleCount),
	}

	m.counters.Range(func(key, value any) bool {
		name := key.(string)
		c := value.(*toggleCounters)

		yes := atomic.SwapInt64(&c.yes, 0)
		no := atomic.SwapInt64(&c.no, 0)

		if yes == 0 && no == 0 {
			c.mu.Lock()
			emptyVariants := len(c.variants) == 0
			c.mu.Unlock()
			if emptyVariants {
				return true
			}
		}

		tc := api.ToggleCount{
			Yes: int32(yes),
			No:  int32(no),
		}

		// we can have a little locking, as a treat. Variants are likely a luke warm path at best
		// until we have evidence that this is a hot path API, I'd like to keep this simple
		// simple here means a local lock per toggle counter while we swap out the variants map
		c.mu.Lock()
		if len(c.variants) > 0 {
			vars := make(map[string]int32, len(c.variants))
			for vName, cnt := range c.variants {
				vars[vName] = int32(cnt)
			}
			tc.Variants = vars

			c.variants = make(map[string]int64)
		}
		c.mu.Unlock()

		bucket.Toggles[name] = tc
		return true
	})

	if len(bucket.Toggles) == 0 {
		return api.Bucket{}, false
	}

	return bucket, true
}

func (m *metrics) sendMetrics() {
	bucket, ok := m.buildBucketAndReset(m.last_close_time)
	if !ok {
		return
	}
	m.last_close_time = time.Now()
	bucket.Stop = time.Now()
	payload := MetricsData{
		AppName:          m.options.appName,
		InstanceID:       m.options.instanceId,
		ConnectionId:     m.options.connectionId,
		Bucket:           bucket,
		SDKVersion:       fmt.Sprintf("%s:%s", clientName, clientVersion),
		PlatformName:     "go",
		PlatformVersion:  runtime.Version(),
		YggdrasilVersion: nil,
		SpecVersion:      specVersion,
	}

	u, _ := m.options.url.Parse("./client/metrics")
	resp, err := m.doPost(u, payload)
	if err != nil {
		m.err(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusMultipleChoices {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
			m.configurationError()
		} else if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			m.backoff()
		}
		m.warn(fmt.Errorf("%s return %d", u.String(), resp.StatusCode))
		// The post failed, re-add the metrics we attempted to send so
		// they are included in the next post.
		m.reinsertBucket(bucket)

		// Set the start time of the current bucket to the one we
		// attempted to send.
		m.last_close_time = bucket.Start

	} else {
		m.successfulPost()
		m.sent <- payload
	}
}

func (m *metrics) doPost(url *url.URL, payload interface{}) (*http.Response, error) {
	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	if err := enc.Encode(payload); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url.String(), &body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(m.ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("UNLEASH-APPNAME", m.options.appName)
	req.Header.Add("UNLEASH-INSTANCEID", m.options.instanceId)
	req.Header.Add("User-Agent", m.options.appName)
	req.Header.Add("Unleash-Interval", fmt.Sprintf("%d", m.options.metricsInterval.Milliseconds()))

	for k, v := range m.options.headers {
		req.Header[k] = v
	}

	return m.options.httpClient.Do(req)
}

func (m *metrics) getOrCreateCounter(name string) *toggleCounters {
	c, ok := m.counters.Load(name)
	if ok {
		return c.(*toggleCounters)
	}

	nc := &toggleCounters{
		variants: make(map[string]int64),
	}
	actual, _ := m.counters.LoadOrStore(name, nc)
	return actual.(*toggleCounters)
}

func (m *metrics) reinsertBucket(bucket api.Bucket) {
	for name, tc := range bucket.Toggles {
		c := m.getOrCreateCounter(name)
		if tc.Yes != 0 {
			atomic.AddInt64(&c.yes, int64(tc.Yes))
		}
		if tc.No != 0 {
			atomic.AddInt64(&c.no, int64(tc.No))
		}

		if len(tc.Variants) > 0 {
			c := m.getOrCreateCounter(name)

			c.mu.Lock()
			if c.variants == nil {
				c.variants = make(map[string]int64, len(tc.Variants))
			}
			for vName, cnt := range tc.Variants {
				if cnt == 0 {
					continue
				}
				c.variants[vName] += int64(cnt)
			}
			c.mu.Unlock()
		}
	}
}

func (m *metrics) add(name string, enabled bool, num int32) {
	if m.options.disableMetrics || num == 0 {
		return
	}
	c := m.getOrCreateCounter(name)
	if enabled {
		atomic.AddInt64(&c.yes, int64(num))
	} else {
		atomic.AddInt64(&c.no, int64(num))
	}
}

func (m *metrics) count(name string, enabled bool) {
	if m.options.disableMetrics {
		return
	}
	m.add(name, enabled, 1)
	m.metricsChannels.count <- metric{Name: name, Enabled: enabled}
}

func (m *metrics) countVariants(name string, enabled bool, variantName string) {
	if m.options.disableMetrics {
		return
	}

	m.add(name, enabled, 1)
	m.metricsChannels.count <- metric{Name: name, Enabled: enabled}

	c := m.getOrCreateCounter(name)

	c.mu.Lock()
	if c.variants == nil {
		c.variants = make(map[string]int64)
	}
	c.variants[variantName]++
	c.mu.Unlock()
}

func (m *metrics) getClientData() ClientData {
	return ClientData{
		m.options.appName,
		m.options.instanceId,
		m.options.connectionId,
		fmt.Sprintf("%s:%s", clientName, clientVersion),
		m.options.strategies,
		m.started,
		int64(m.options.metricsInterval.Seconds()),
		runtime.Version(),
		"go",
		nil,
		specVersion,
	}
}
