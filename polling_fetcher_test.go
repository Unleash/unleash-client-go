package unleash

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/h2non/gock"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type NoOpStorage struct{}

func (s *NoOpStorage) Persist(features *api.FeatureResponse) error {
	return nil
}

func (s *NoOpStorage) Load() (*api.FeatureResponse, error) {
	return &api.FeatureResponse{}, nil
}

func (s *NoOpStorage) Init(backupPath, appName string) {}

// TestPollingFetcher_GetFeaturesFail tests that OnReady isn't fired unless
// /client/features has returned successfully.
func TestPollingFetcher_GetFeaturesFail(t *testing.T) {
	assert := assert.New(t)
	featuresCalls := make(chan int, 10)
	var sendStatus200 int32
	prevStatus := 0
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method + " " + req.URL.Path {
		case "POST /client/register":
		case "GET /client/features":
			status200 := atomic.LoadInt32(&sendStatus200) == 1
			status := 0
			if status200 {
				status = 200
				rw.WriteHeader(200)
				writeJSON(rw, api.FeatureResponse{})
			} else {
				status = 400
				rw.WriteHeader(400)
			}
			if status != prevStatus {
				featuresCalls <- status
				prevStatus = status
			}
		case "POST /client/metrics":
		default:
			t.Fatalf("Unexpected request: %+v", req)
		}
	}))
	defer srv.Close()

	ready := make(chan struct{})
	mockListener := &MockedListener{}
	mockListener.On("OnReady").Run(func(args mock.Arguments) { close(ready) }).Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnError", mock.MatchedBy(func(e error) bool {
		return strings.HasSuffix(e.Error(), "/client/features returned status code 400")
	})).Return()
	mockListener.On("OnSent", mock.AnythingOfType("MetricsData")).Return()
	client, err := NewClient(
		WithUrl(srv.URL),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
		WithRefreshInterval(time.Millisecond),
	)
	assert.Nil(err, "client should not return an error")

	assert.Equal(400, <-featuresCalls)
	select {
	case <-ready:
		t.Fatal("client is ready but it shouldn't be")
	case <-time.NewTimer(time.Second).C:
	}

	atomic.StoreInt32(&sendStatus200, 1)
	assert.Equal(200, <-featuresCalls)

	select {
	case <-ready:
	case <-time.NewTimer(time.Second).C:
		t.Fatal("client isn't ready but should be")
	}
	client.Close()
}

func TestPollingFetcher_OnUpdateCalledWhenFeaturesChangeOnly(t *testing.T) {
	assert := assert.New(t)
	featuresCalls := make(chan int, 10)
	var sendStatus304 int32
	prevStatus := 0
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method + " " + req.URL.Path {
		case "POST /client/register":
		case "GET /client/features":
			status304 := atomic.LoadInt32(&sendStatus304) == 1
			status := 0
			if status304 {
				status = 304
				rw.WriteHeader(304)
			} else {
				status = 200
				rw.WriteHeader(200)
				writeJSON(rw, api.FeatureResponse{})
			}
			if status != prevStatus {
				featuresCalls <- status
				prevStatus = status
			}
		case "POST /client/metrics":
		default:
			t.Fatalf("Unexpected request: %+v", req)
		}
	}))
	defer srv.Close()

	update := make(chan bool)
	mockListener := &MockedListener{}
	mockListener.On("OnUpdate").Run(func(args mock.Arguments) { update <- true }).Return()
	mockListener.On("OnReady").Run(func(args mock.Arguments) {}).Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnSent", mock.AnythingOfType("MetricsData")).Return()
	client, err := NewClient(
		WithUrl(srv.URL),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
		WithRefreshInterval(time.Millisecond),
		WithDisableMetrics(true),
	)
	assert.Nil(err, "client should not return an error")

	assert.Equal(200, <-featuresCalls)

	select {
	case <-update:
	case <-time.NewTimer(time.Second).C:
		t.Fatal("client did not call OnUpdate")
	}

	atomic.StoreInt32(&sendStatus304, 1)
	assert.Equal(304, <-featuresCalls)

	select {
	case <-update:
		t.Fatal("client called OnUpdate but it shouldn't have")
	case <-time.NewTimer(time.Second).C:
	}

	close(update)
	client.Close()
}

func TestPollingFetcher_ParseAPIResponse(t *testing.T) {
	assert := assert.New(t)
	data := []byte(`{
			"version": 2,
			"features": [
				{
					"strategies": [],
					"impressionData": false,
					"enabled": false,
					"name": "my-feature",
					"description": "",
					"project": "default",
					"stale": false,
					"type": "release",
					"variants": []
				},
				{
					"strategies": [],
					"impressionData": false,
					"enabled": false,
					"name": "my-new-feature",
					"description": "",
					"project": "default",
					"stale": false,
					"type": "release",
					"variants": []
				}
			],
			"query": {
				"inlineSegmentConstraints": true
			}
		}`)

	reader := bytes.NewReader(data)
	dec := json.NewDecoder(reader)

	var response api.FeatureResponse

	err := dec.Decode(&response)

	assert.Nil(err)

	assert.Equal(2, len(response.Features))
	assert.Equal(0, len(response.Segments))
}

func TestPollingFetcher_backs_off_on_http_statuses(t *testing.T) {
	a := assert.New(t)
	testCases := []struct {
		statusCode int
		errorCount float64
	}{
		{401, 10},
		{403, 10},
		{404, 10},
		{429, 1},
		{500, 1},
		{502, 1},
		{503, 1},
	}
	defer gock.Off()
	for _, tc := range testCases {
		gock.New(mockerServer).
			Get("/client/features").
			Reply(tc.statusCode)
		serverURL, err := url.Parse(mockerServer)
		a.Nil(err)
		errChannels := errorChannels{
			errors:   make(chan error, 10),
			warnings: make(chan error, 10),
		}
		fetcherChannels := fetcherChannels{
			errorChannels: errChannels,
			ready:         make(chan bool, 1),
			update:        make(chan bool, 1),
		}
		fetcher := newPollingFetcher(
			fetcherOptions{
				url:             *serverURL,
				appName:         mockAppName,
				instanceId:      mockInstanceId,
				refreshInterval: time.Millisecond * 15,
				storage:         &NoOpStorage{},
				httpClient:      http.DefaultClient,
				headers:         make(http.Header),
			},
			fetcherChannels,
		)
		fetcher.start()
		time.Sleep(20 * time.Millisecond)
		fetcher.stop()
		a.Equal(tc.errorCount, fetcher.errors)
	}
}

func TestPollingFetcher_back_offs_are_gradually_reduced_on_success(t *testing.T) {
	a := assert.New(t)
	defer gock.Off()
	gock.New(mockerServer).
		Get("/client/features").
		Times(4).
		Reply(429)
	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		BodyString(`{ "version": 2, "features": []}`)
	serverURL, err := url.Parse(mockerServer)
	a.Nil(err)
	errChannels := errorChannels{
		errors:   make(chan error, 10),
		warnings: make(chan error, 10),
	}
	fetcherChannels := fetcherChannels{
		errorChannels: errChannels,
		ready:         make(chan bool, 1),
		update:        make(chan bool, 1),
	}
	fetcher := newPollingFetcher(
		fetcherOptions{
			url:             *serverURL,
			appName:         mockAppName,
			instanceId:      mockInstanceId,
			refreshInterval: time.Millisecond * 10,
			storage:         &NoOpStorage{},
			httpClient:      http.DefaultClient,
			headers:         make(http.Header),
		},
		fetcherChannels,
	)
	fetcher.start()
	select {
	case <-fetcherChannels.ready:
	case <-time.NewTimer(time.Second).C:
		t.Fatal("fetcher isn't ready but should be")
	}
	fetcher.stop()
	a.Equal(float64(3), fetcher.errors) // 4 failures, and then one success, should reduce error count to 3
}
