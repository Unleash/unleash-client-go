package unleash

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Unleash/unleash-client-go/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/h2non/gock.v1"
)

func TestMetrics_RegisterInstance(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	gock.New(mockerServer).
		Get("/features").
		Reply(200).
		JSON(api.FeatureResponse{})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)
	defer func() {
		client.Close()
	}()
	time.Sleep(1 * time.Second)

	assert.Nil(err, "client should not return an error")

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestMetrics_DoPost(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	gock.New(mockerServer).
		Post("").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	gock.New(mockerServer).
		Get("/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Response: api.Response{
				Version: 1,
			},
			Features: []api.Feature{
				{
					Name:    "foo",
					Enabled: true,
					Strategies: []api.Strategy{
						{
							Name: "default",
						},
					},
				},
			},
		})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", mock.AnythingOfType("string"), mock.AnythingOfType("bool"))
	mockListener.On("OnSent", mock.AnythingOfType("MetricsData"))

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithRefreshInterval(5*time.Second),
		WithListener(mockListener),
	)
	<-client.Ready()
	defer func() {
		client.Close()
	}()
	assert.Nil(err, "client should not return an error")

	m := client.metrics

	serverURL, _ := url.Parse(mockerServer)
	res, err := m.doPost(serverURL, &struct{}{})
	fmt.Println("checking results...")
	assert.Nil(err, "doPost should not return an error")
	assert.Equal(200, res.StatusCode, "statusCode should be 200")
	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestMetrics_SendMetrics(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	gock.New(mockerServer).
		Post("/client/metrics").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Times(5).
		Reply(200)

	gock.New(mockerServer).
		Get("/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Response: api.Response{
				Version: 1,
			},
			Features: []api.Feature{
				{
					Name:    "foo",
					Enabled: true,
					Strategies: []api.Strategy{
						{
							Name: "default",
						},
					},
				},
			},
		})
	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", mock.AnythingOfType("string"), mock.AnythingOfType("bool"))
	mockListener.On("OnSent", mock.AnythingOfType("MetricsData"))

	fmt.Printf("mock http with transport: %T\n", http.DefaultTransport)
	for _, m := range gock.GetAll() {
		r := m.Request()
		fmt.Printf("mock http request: '%s %s'\n", r.Method, r.URLStruct.String())
	}
	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithMetricsInterval(200*time.Millisecond),
		WithListener(mockListener),
		WithRefreshInterval(200*time.Second), // make sure refresh interval is large enough
	)
	assert.Nil(err, "client should not return an error")
	defer client.Close()

	for i := 0; i < 20; i++ {
		client.IsEnabled("foo")

		if i%5 == 0 {
			time.Sleep(1 * time.Second)
		}
	}

	time.Sleep(1 * time.Second)

	mockListener.AssertNumberOfCalls(t, "OnRegistered", 1)
	mockListener.AssertNumberOfCalls(t, "OnReady", 1)
	mockListener.AssertNumberOfCalls(t, "OnCount", 20)
	mockListener.AssertNumberOfCalls(t, "OnSent", 5)

	assert.True(gock.IsDone(), "there should be no more mocks")
}
