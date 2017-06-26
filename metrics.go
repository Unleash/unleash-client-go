package unleash

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Unleash/unleash-client-go/internal/api"
	"net/http"
	"time"
	"net/url"
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
	options      metricsOptions
	started      time.Time
	bucket       api.Bucket
	countChannel chan metric
	stopped      chan bool
	timer        *time.Timer
}

func newMetrics(options metricsOptions, channels metricsChannels) *metrics {
	m := &metrics{
		metricsChannels: channels,
		options:         options,
		started:         time.Now(),
		countChannel:    make(chan metric),
		stopped:         make(chan bool),
	}

	if m.options.httpClient == nil {
		m.options.httpClient = http.DefaultClient
	}

	m.resetBucket()

	if m.options.metricsInterval > 0 {
		m.startTimer()
		m.registerInstance()
		go m.sync()
	}

	return m
}

func (m *metrics) Close() error {
	m.stop()
	return nil
}

func (m *metrics) startTimer() {
	if m.options.disableMetrics {
		return
	}

	m.timer = time.NewTimer(m.options.metricsInterval)
}

func (m *metrics) stop() {
	if !m.timer.Stop() {
		<-m.timer.C
	}
	m.stopped <- true
}

func (m *metrics) sync() {
	for {
		select {
		case mc := <-m.countChannel:
			t, exists := m.bucket.Toggles[mc.Name]
			if !exists {
				t = api.ToggleCount{}
			}
			if mc.Enabled {
				t.Yes++
			} else {
				t.No++
			}
			m.bucket.Toggles[mc.Name] = t
		case <-m.timer.C:
			m.sendMetrics()
		case <-m.stopped:
			m.options.disableMetrics = true
			return
		}
	}

}

func (m *metrics) registerInstance() {
	if m.options.disableMetrics {
		return
	}

	u, _ := m.options.url.Parse("./client/register")
	payload := m.getClientData()
	resp, err := m.doPost(u, payload)

	if err != nil {
		m.err(err)
		return
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusMultipleChoices {
		m.warn(fmt.Errorf("%s return %d", u.String(), resp.StatusCode))
	}

	m.registered <- payload
}

func (m *metrics) sendMetrics() {
	if m.options.disableMetrics {
		return
	}

	if m.bucket.IsEmpty() {
		m.resetBucket()
		m.startTimer()
		return
	}

	u, _ := m.options.url.Parse("./client/metrics")
	payload := m.getPayload()
	m.startTimer()
	resp, err := m.doPost(u, payload)

	if err != nil {
		m.err(err)
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		m.warn(fmt.Errorf("%s return 404, stopping metrics", u.String()))
		m.stop()
		return
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode > http.StatusMultipleChoices {
		m.warn(fmt.Errorf("%s return %d", u.String(), resp.StatusCode))
	}

	m.sent <- payload
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Unleash-Appname", m.options.appName)
	req.Header.Add("Unleash-Instanceid", m.options.instanceId)
	req.Header.Add("User-Agent", m.options.appName)

	for k, v := range m.options.customHeaders {
		req.Header[k] = v
	}

	return m.options.httpClient.Do(req)
}

func (m metrics) count(name string, enabled bool) {
	if m.options.disableMetrics {
		return
	}
	m.countChannel <- metric{name, enabled}
}

func (m *metrics) resetBucket() {
	m.bucket = api.Bucket{
		Start:   time.Now(),
		Toggles: map[string]api.ToggleCount{},
	}
}

func (m *metrics) closeBucket() {
	m.bucket.Stop = time.Now()
}

func (m *metrics) getPayload() MetricsData {
	m.closeBucket()
	metricsData := m.getMetricsData()
	m.resetBucket()
	return metricsData
}

func (m metrics) getClientData() ClientData {
	return ClientData{
		m.options.appName,
		m.options.instanceId,
		m.options.strategies,
		m.started,
		int64(m.options.metricsInterval.Seconds()),
	}
}

func (m metrics) getMetricsData() MetricsData {
	return MetricsData{
		m.options.appName,
		m.options.instanceId,
		m.bucket,
	}
}
