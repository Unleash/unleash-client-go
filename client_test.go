package unleash

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Unleash/unleash-client-go/internal/api"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gock "gopkg.in/h2non/gock.v1"
)

func TestGetFeaturesByPattern(t *testing.T) {
	// given
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	gock.New(mockerServer).
		Get("/features").
		Reply(200).
		JSON(api.FeatureResponse{
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
				{
					Name:    "foo.bar",
					Enabled: true,
					Strategies: []api.Strategy{
						{
							Name: "default",
						},
					},
				},
				{
					Name:    "bar",
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

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)
	require.Nil(t, err)
	defer func() {
		client.Close()
	}()
	<-client.Ready()
	// when
	features := client.GetFeaturesByPattern("foo")
	// then
	// just check that result contains feature names
	featuresStr := make([]string, len(features))
	for i, f := range features {
		featuresStr[i] = f.Name
	}
	require.NotEmpty(t, featuresStr)
	assert.Contains(t, featuresStr, "foo")
	assert.Contains(t, featuresStr, "foo.bar")
	assert.NotContains(t, featuresStr, "bar")
}

func TestGetFeature(t *testing.T) {
	// given
	// given
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	gock.New(mockerServer).
		Get("/features").
		Reply(200).
		JSON(api.FeatureResponse{
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

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)
	require.Nil(t, err)
	defer func() {
		client.Close()
	}()
	<-client.Ready()
	// when
	feature := client.GetFeature("foo")
	// then
	assert.NotNil(t, feature)
}
