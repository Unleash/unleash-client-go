package unleash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Unleash/unleash-client-go/v3/internal/api"
)

// MetricsData represents the data sent to the unleash server.
type MetricsData struct {
	// AppName is the name of the application.
	AppName string `json:"appName"`

	// InstanceID is the instance identifier.
	InstanceID string `json:"instanceId"`

	// Bucket is the payload data sent to the server.
	Bucket api.Bucket `json:"bucket"`
}

// ClientData represents the data sent to the unleash during registration.
type ClientData struct {
	// AppName is the name of the application.
	AppName string `json:"appName"`

	// InstanceID is the instance identifier.
	InstanceID string `json:"instanceId"`

	// Optional field that describes the sdk version (name:version)
	SDKVersion string `json:"sdkVersion"`

	// Strategies is a list of names of the strategies supported by the client.
	Strategies []string `json:"strategies"`

	// Started indicates the time at which the client was created.
	Started time.Time `json:"started"`

	// Interval specifies the time interval (in ms) that the client is using for refreshing
	// feature toggles.
	Interval int64 `json:"interval"`
}

type metric struct {
	// Name is the name of the feature toggle.
	Name string

	// Enabled indicates whether the feature was enabled or not.
	Enabled bool
}

type metrics struct {
	metricsChannels
	options  metricsOptions
	started  time.Time
	bucketMu sync.Mutex
	bucket   api.Bucket
	ticker   *time.Ticker
	close    chan struct{}
	closed   chan struct{}
	ctx      context.Context
	cancel   func()
}

func newMetrics(options metricsOptions, channels metricsChannels) *metrics {
	m := &metrics{
		metricsChannels: channels,
		options:         options,
		started:         time.Now(),
		close:           make(chan struct{}),
		closed:          make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.ctx = ctx
	m.cancel = cancel

	if m.options.httpClient == nil {
		m.options.httpClient = http.DefaultClient
	}

	m.resetBucket()
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
			m.sendMetrics()
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

func (m *metrics) sendMetrics() {
	m.bucketMu.Lock()
	bucket := m.resetBucket()
	m.bucketMu.Unlock()
	if bucket.IsEmpty() {
		return
	}
	bucket.Stop = time.Now()
	payload := MetricsData{
		AppName:    m.options.appName,
		InstanceID: m.options.instanceId,
		Bucket:     bucket,
	}

	u, _ := m.options.url.Parse("./client/metrics")
	resp, err := m.doPost(u, payload)
	if err != nil {
		m.err(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusMultipleChoices {
		m.warn(fmt.Errorf("%s return %d", u.String(), resp.StatusCode))
		// The post failed, re-add the metrics we attempted to send so
		// they are included in the next post.
		for name, tc := range bucket.Toggles {
			m.add(name, true, tc.Yes)
			m.add(name, false, tc.No)
		}

		m.bucketMu.Lock()
		// Set the start time of the current bucket to the one we
		// attempted to send.
		m.bucket.Start = bucket.Start
		m.bucketMu.Unlock()
	} else {
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

	for k, v := range m.options.customHeaders {
		req.Header[k] = v
	}

	return m.options.httpClient.Do(req)
}

func (m *metrics) add(name string, enabled bool, num int32) {
	if m.options.disableMetrics || num == 0 {
		return
	}
	m.bucketMu.Lock()
	defer m.bucketMu.Unlock()
	t, exists := m.bucket.Toggles[name]
	if !exists {
		t = api.ToggleCount{
			Variants: map[string]int32{},
		}
	}
	if enabled {
		t.Yes += num
	} else {
		t.No += num
	}
	m.bucket.Toggles[name] = t
}

func (m *metrics) count(name string, enabled bool) {
	if m.options.disableMetrics {
		return
	}
	m.add(name, enabled, 1)
	m.metricsChannels.count <- metric{Name: name, Enabled: enabled}
}

func (m *metrics) countVariants(name string, variantName string) {
	if m.options.disableMetrics {
		return
	}
	
	t, _ := m.bucket.Toggles[name]
	if len(t.Variants) == 0 {
		t.Variants = make(map[string]int32)
	} 

	if _ , ok := t.Variants[variantName]; !ok {
		t.Variants[variantName] = 1
	} else {
		t.Variants[variantName] += 1
	}
	m.bucket.Toggles[name] = t
}

func (m *metrics) resetBucket() api.Bucket {
	prev := m.bucket
	m.bucket = api.Bucket{
		Start:   time.Now(),
		Toggles: map[string]api.ToggleCount{},
	}
	return prev
}

func (m *metrics) getClientData() ClientData {
	return ClientData{
		m.options.appName,
		m.options.instanceId,
		fmt.Sprintf("%s:%s", clientName, clientVersion),
		m.options.strategies,
		m.started,
		int64(m.options.metricsInterval.Seconds()),
	}
}
