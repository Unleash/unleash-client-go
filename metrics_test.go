package unleash

import (
	"github.com/Unleash/unleash-client-go/v3/internal/api"
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
		Get("/client/features").
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
	defer client.Close()

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
	defer client.Close()

	assert.Nil(err, "client should not return an error")

	m := client.metrics

	serverUrl, _ := url.Parse(mockerServer)
	res, err := m.doPost(serverUrl, &struct{}{})

	assert.Nil(err, "doPost should not return an error")
	assert.Equal(200, res.StatusCode, "statusCode should be 200")
	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestMetrics_DisabledMetrics(t *testing.T) {
	assert := assert.New(t)
	defer gock.Off()

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()

	client, err := NewClient(
		WithUrl(mockerServer),
		WithDisableMetrics(true),
		WithMetricsInterval(100*time.Millisecond),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	assert.Nil(err, "client should not return an error")

	client.WaitForReady()

	client.IsEnabled("foo")
	client.IsEnabled("bar")
	client.IsEnabled("baz")

	time.Sleep(300 * time.Millisecond)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

