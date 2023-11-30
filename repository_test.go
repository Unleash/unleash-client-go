package unleash

import (
	"bytes"
	"encoding/json"
	"gopkg.in/h2non/gock.v1"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Unleash/unleash-client-go/v4/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestRepository_GetFeaturesFail tests that OnReady isn't fired unless
// /client/features has returned successfully.
func TestRepository_GetFeaturesFail(t *testing.T) {
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

func TestRepository_ParseAPIResponse(t *testing.T) {
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

func TestRepository_backs_off_on_http_statuses(t *testing.T) {
	a := assert.New(t)
	testCases := []struct {
		statusCode int
		errorCount float64
	}{
		{ 401, 10},
		{ 403, 10},
		{ 404, 10},
		{ 429, 1},
		{ 500, 1},
		{ 502, 1},
		{ 503, 1},
	}
	defer gock.Off()
	for _, tc := range testCases {
		gock.New(mockerServer).
			Get("/client/features").
			Reply(tc.statusCode)
		client, err := NewClient(
			WithUrl(mockerServer),
			WithAppName(mockAppName),
			WithDisableMetrics(true),
			WithInstanceId(mockInstanceId),
			WithRefreshInterval(time.Millisecond * 15),
		)
		a.Nil(err)
		time.Sleep(20 * time.Millisecond)
		a.Equal(tc.errorCount, client.repository.errors)
		err = client.Close()
		a.Nil(err)
	}
}

func TestRepository_back_offs_are_gradually_reduced_on_success(t *testing.T) {
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
	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithDisableMetrics(true),
		WithInstanceId(mockInstanceId),
		WithRefreshInterval(time.Millisecond * 10),
	)
	a.Nil(err)
	client.WaitForReady()
	a.Equal(float64(3), client.repository.errors) // 4 failures, and then one success, should reduce error count to 3
	err = client.Close()
	a.Nil(err)
}