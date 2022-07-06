package unleash

import (
	"time"

	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"testing"

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
				t.Errorf("Unexpected error: %v", e)
				return
			case w := <-client.Warnings():
				t.Errorf("Unexpected warning: %v", w)
				return
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

func TestClient_ListFeatures(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	features := []api.Feature{
		{
			Name:        "feature-name",
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "feature-strategy",
			Strategies: []api.Strategy{
				{
					Id:   1,
					Name: "strategy-name",
					Constraints: []api.Constraint{
						{
							ContextName: "context-name",
							Operator:    api.OperatorIn,
							Values:      []string{"constraint-value-1", "constraint-value-2"},
						},
					},
					Parameters: map[string]interface{}{
						"strategy-param-1": "strategy-value-1",
					},
				},
			},
			Parameters: map[string]interface{}{
				"feature-param-1": "feature-value-1",
			},
		},
	}

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Features: features,
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

	assert.NoError(err)

	client.WaitForReady()

	require.Equal(t, features, client.ListFeatures())
}

func TestClientWithProjectName(t *testing.T) {
	assert := assert.New(t)
	projectName := "myProject"
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	gock.New(mockerServer).
		Get("/client/features").
		MatchParam("project", projectName).
		Reply(200).
		JSON(api.FeatureResponse{})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithProjectName(projectName),
		WithListener(mockListener),
	)

	client.WaitForReady()

	assert.NoError(err)
	assert.Equal(client.options.projectName, projectName)
	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClientWithoutProjectName(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
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

	client.WaitForReady()

	assert.NoError(err)
	assert.Equal(client.options.projectName, "")
	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClientWithVariantContext(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	features := []api.Feature{
		{
			Name:        "feature-name",
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "feature-strategy",
			Strategies: []api.Strategy{
				{
					Id:   1,
					Name: "default",
					Constraints: []api.Constraint{
						{
							ContextName: "custom-id",
							Operator:    api.OperatorIn,
							Values:      []string{"custom-ctx"},
						},
					},
					Parameters: map[string]interface{}{
						"strategy-param-1": "strategy-value-1",
					},
				},
			},
			Parameters: map[string]interface{}{
				"feature-param-1": "feature-value-1",
			},
			Variants: []api.VariantInternal{
				{
					Variant: api.Variant{
						Name:    "custom-variant",
						Payload: api.Payload{},
						Enabled: true,
					},
					Weight:     100,
					WeightType: "",
					Overrides:  nil,
				},
			},
		},
	}

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Features: features,
			Segments: []api.Segment{},
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

	assert.NoError(err)

	client.WaitForReady()

	defaultVariant := client.GetVariant("feature-name")
	assert.Equal(api.GetDefaultVariant(), defaultVariant)
	variant := client.GetVariant("feature-name", WithVariantContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx"},
	}))
	assert.Equal("custom-variant", variant.Name)
	assert.True(gock.IsDone(), "there should be no more mocks")
}
