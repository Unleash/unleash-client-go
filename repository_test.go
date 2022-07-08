package unleash

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Unleash/unleash-client-go/v3/api"
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

// func TestRepository_SetSegmentsMap(t *testing.T) {
// 	parsedUrl, err := url.Parse("http://foo.com")
// 	if err != nil {
// 		return;
// 	}
// 	options := repositoryOptions{
// 		appName: "test-app",
// 		instanceId: "my-instance",
// 		projectName: "default",
// 		url: *parsedUrl,
// 		backupPath: "",
// 		refreshInterval: 1,
// 		segments: map[int]api.Segment{},
// 		storage: &DefaultStorage{},
// 		customHeaders: http.Header{},
// 		httpClient: &http.Client{},
// 	}

//    repository := newRepository(options, repositoryChannels{errorChannels: errorChannels{
// 	errors: make(chan error),
// 	warnings: make(chan error),
//    }, ready: make(chan string))

//    fmt.Print(repository)
// }
