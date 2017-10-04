package unleash

import (
	"github.com/Unleash/unleash-client-go/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/h2non/gock.v1"
	"net/url"
	"testing"
	"time"
)

func TestMetrics_RegisterInstance(t *testing.T) {
	assert := assert.New(t)
	defer gock.Off()

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

	_, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	time.Sleep(1 * time.Second)

	assert.Nil(err, "client should not return an error")

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestMetrics_DoPost(t *testing.T) {
	assert := assert.New(t)
	defer gock.Off()

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	gock.New(mockerServer).
		Post("").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
	)

	assert.Nil(err, "client should not return an error")

	m := client.metrics

	serverUrl, _ := url.Parse(mockerServer)
	res, err := m.doPost(serverUrl, &struct{}{})

	assert.Nil(err, "doPost should not return an error")
	assert.Equal(200, res.StatusCode, "statusCode should be 200")
	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestMetrics_SendMetrics(t *testing.T) {
	assert := assert.New(t)
	defer gock.Off()

	gock.New(mockerServer).
		Post("/client/register").
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

	gock.New(mockerServer).
		Post("/client/metrics").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Times(5).
		Reply(200)

	mockListener := &MockedListener{}

	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", mock.AnythingOfType("string"), mock.AnythingOfType("bool"))
	mockListener.On("OnSent", mock.AnythingOfType("MetricsData"))

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithMetricsInterval(200*time.Millisecond),
		WithListener(mockListener),
	)
	assert.Nil(err, "client should not return an error")

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
