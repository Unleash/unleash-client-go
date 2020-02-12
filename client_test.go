package unleash

import (
	"testing"

	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/stretchr/testify/mock"

	"github.com/Unleash/unleash-client-go/v3/internal/api"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func TestClientWithoutListener(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{})

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
	)
	assert.Nil(err, "client should not return an error")

	go func() {
		for {
			select {
			case e := <-client.Errors():
				t.Fatalf("Unexpected error: %v", e)
			case w := <-client.Warnings():
				t.Fatalf("Unexpected warning: %v", w)
			case <-client.Count():
			case <-client.Sent():
			}
		}
	}()
	<-client.Registered()
	<-client.Ready()
	client.Close()
	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClient_WithFallbackFunc(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{})

	feature := "does_not_exist"

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, true).Return()

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	assert.NoError(err)

	client.WaitForReady()

	fallback := func(f string, ctx *context.Context) bool {
		return f == feature
	}

	isEnabled := client.IsEnabled("does_not_exist", WithFallbackFunc(fallback))
	assert.True(isEnabled)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClientWithSqliteDatabase(t *testing.T) {
	assert := assert.New(t)

	_, _, path := buildTestSqliteRepository(assert)

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithDatabasePath(path),
		WithDisableMetrics(true),
	)
	assert.Nil(err, "client should not return an error")

	go func() {
		for {
			select {
			case e := <-client.Errors():
				t.Fatalf("Unexpected error: %v", e)
			case w := <-client.Warnings():
				t.Fatalf("Unexpected warning: %v", w)
			case <-client.Count():
			case <-client.Sent():
			}
		}
	}()
	<-client.Ready()

	assert.True(client.IsEnabled("dummy.feature1"))

	assert.False(client.IsEnabled("dummy.feature2"))
	client.Close()
}
