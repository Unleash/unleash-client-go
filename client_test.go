package unleash

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5/api"
	"github.com/Unleash/unleash-go-sdk/v5/context"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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

	err = client.Close()
	assert.Nil(err)
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
	mockListener.On("OnError").Return()

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

func TestClient_WithResolver(t *testing.T) {
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

	const feature = "some_special_value"

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, true).Return()
	mockListener.On("OnError").Return()

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	assert.NoError(err)

	client.WaitForReady()

	resolver := func(featureName string) *api.Feature {
		if featureName == feature {
			return &api.Feature{
				Name:        "some_special_value-resolved",
				Description: "",
				Enabled:     true,
				Strategies: []api.Strategy{
					{
						Id:   1,
						Name: "default",
					},
				},
				CreatedAt:  time.Time{},
				Strategy:   "default-strategy",
				Parameters: nil,
				Variants:   nil,
			}
		} else {
			t.Fatalf("the feature name passed %s was not the expected one %s", featureName, feature)
			return nil
		}
	}

	isEnabled := client.IsEnabled(feature, WithResolver(resolver))
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
	mockListener.On("onError").Return()

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
	mockListener.On("OnError").Return()

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
	mockListener.On("OnError").Return()

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
	mockListener.On("OnCount", mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnError", mock.AnythingOfType("*errors.errorString"))

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

	variantFromResolver := client.GetVariant("feature-name", WithVariantContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx"},
	}), WithVariantResolver(func(featureName string) *api.Feature {
		if featureName == features[0].Name {
			return &features[0]
		} else {
			t.Fatalf("the feature name passed %s was not the expected one %s", featureName, features[0].Name)
			return nil
		}
	}))

	assert.Equal("custom-variant", variantFromResolver.Name)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClient_WithSegment(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	feature := "feature-segment"
	features := []api.Feature{
		{
			Name:        feature,
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "feature-strategy",
			Strategies: []api.Strategy{
				{
					Id:          1,
					Name:        "default",
					Constraints: []api.Constraint{},
					Parameters:  map[string]interface{}{},
					Segments:    []int{1},
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
			Segments: []api.Segment{
				{Id: 1, Constraints: []api.Constraint{
					{
						ContextName: "custom-id",
						Operator:    api.OperatorIn,
						Values:      []string{"custom-ctx"},
					}}},
			}})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, true).Return()
	mockListener.On("OnError").Return()

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	assert.NoError(err)
	client.WaitForReady()

	isEnabled := client.IsEnabled(feature, WithContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx"},
	}))

	assert.True(isEnabled)

	isEnabledWithResolver := client.IsEnabled(feature, WithContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx"},
	}), WithResolver(func(featureName string) *api.Feature {
		if featureName == features[0].Name {
			return &features[0]
		} else {
			t.Fatalf("the feature name passed %s was not the expected one %s", featureName, features[0].Name)
			return nil
		}
	}))

	assert.True(isEnabledWithResolver)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClient_WithNonExistingSegment(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	feature := "feature-segment-non-existing"
	features := []api.Feature{
		{
			Name:        feature,
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "feature-strategy",
			Strategies: []api.Strategy{
				{
					Id:          1,
					Name:        "default",
					Constraints: []api.Constraint{},
					Parameters:  map[string]interface{}{},
					Segments:    []int{1},
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
			Segments: []api.Segment{}})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, false).Return()
	mockListener.On("OnError", mock.AnythingOfType("*errors.errorString"))

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	assert.NoError(err)

	client.WaitForReady()

	isEnabled := client.IsEnabled(feature, WithContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx"},
	}))

	assert.False(isEnabled)

	isEnabledWithResolver := client.IsEnabled(feature, WithContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx"},
	}), WithResolver(func(featureName string) *api.Feature {
		if featureName == features[0].Name {
			return &features[0]
		} else {
			t.Fatalf("the feature name passed %s was not the expected one %s", featureName, features[0].Name)
			return nil
		}
	}))

	assert.False(isEnabledWithResolver)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClient_WithMultipleSegments(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	feature := "feature-segment-multiple"
	features := []api.Feature{
		{
			Name:        feature,
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "feature-strategy",
			Strategies: []api.Strategy{
				{
					Id:          1,
					Name:        "default",
					Constraints: []api.Constraint{},
					Parameters:  map[string]interface{}{},
					Segments:    []int{1, 4, 6, 2},
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
			Segments: []api.Segment{
				{Id: 1, Constraints: []api.Constraint{
					{
						ContextName: "custom-id",
						Operator:    api.OperatorIn,
						Values:      []string{"custom-ctx"},
					}}},
				{Id: 2, Constraints: []api.Constraint{
					{
						ContextName: "semver",
						Operator:    api.OperatorSemverGt,
						Value:       "3.2.1",
					}}},
				{Id: 4, Constraints: []api.Constraint{
					{
						ContextName: "age",
						Operator:    api.OperatorNumEq,
						Value:       "18",
					}}},
				{Id: 6, Constraints: []api.Constraint{
					{
						ContextName: "domain",
						Operator:    api.OperatorStrStartsWith,
						Values:      []string{"unleash"},
					}}},
			}})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, true).Return()
	mockListener.On("OnError").Return()

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	assert.NoError(err)
	client.WaitForReady()

	isEnabled := client.IsEnabled(feature, WithContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx", "semver": "3.2.2", "age": "18", "domain": "unleashtest"},
	}))

	assert.True(isEnabled)

	isEnabledWithResolver := client.IsEnabled(feature, WithContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx", "semver": "3.2.2", "age": "18", "domain": "unleashtest"},
	}), WithResolver(func(featureName string) *api.Feature {
		if featureName == features[0].Name {
			return &features[0]
		} else {
			t.Fatalf("the feature name passed %s was not the expected one %s", featureName, features[0].Name)
			return nil
		}
	}))

	assert.True(isEnabledWithResolver)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClient_VariantShouldRespectConstraint(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	feature := "feature-segment-multiple"
	features := []api.Feature{
		{
			Name:        feature,
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "feature-strategy",
			Strategies: []api.Strategy{
				{
					Id:          1,
					Name:        "default",
					Constraints: []api.Constraint{},
					Parameters:  map[string]interface{}{},
					Segments:    []int{1, 4, 6, 2},
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
			Segments: []api.Segment{
				{Id: 1, Constraints: []api.Constraint{
					{
						ContextName: "custom-id",
						Operator:    api.OperatorIn,
						Values:      []string{"custom-ctx"},
					}}},
				{Id: 2, Constraints: []api.Constraint{
					{
						ContextName: "semver",
						Operator:    api.OperatorSemverGt,
						Value:       "3.2.1",
					}}},
				{Id: 4, Constraints: []api.Constraint{
					{
						ContextName: "age",
						Operator:    api.OperatorNumEq,
						Value:       "18",
					}}},
				{Id: 6, Constraints: []api.Constraint{
					{
						ContextName: "domain",
						Operator:    api.OperatorStrStartsWith,
						Values:      []string{"unleash"},
					}}},
			}})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, true).Return()
	mockListener.On("OnError").Return()

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	assert.NoError(err)
	client.WaitForReady()

	variant := client.GetVariant(feature, WithVariantContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx", "semver": "3.2.2", "age": "18", "domain": "unleashtest"},
	}))

	assert.True(variant.Enabled)

	assert.True(variant.FeatureEnabled)

	variantFromResolver := client.GetVariant(feature, WithVariantContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx", "semver": "3.2.2", "age": "18", "domain": "unleashtest"},
	}), WithVariantResolver(func(featureName string) *api.Feature {
		if featureName == features[0].Name {
			return &features[0]
		} else {
			t.Fatalf("the feature name passed %s was not the expected one %s", featureName, features[0].Name)
			return nil
		}
	}))

	assert.True(variantFromResolver.Enabled)

	assert.True(variantFromResolver.FeatureEnabled)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClient_VariantShouldFailWhenSegmentConstraintsDontMatch(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	feature := "feature-segment-multiple"
	features := []api.Feature{
		{
			Name:        feature,
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "feature-strategy",
			Strategies: []api.Strategy{
				{
					Id:          1,
					Name:        "default",
					Constraints: []api.Constraint{},
					Parameters:  map[string]interface{}{},
					Segments:    []int{1, 4, 6, 2},
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
			Segments: []api.Segment{
				{Id: 1, Constraints: []api.Constraint{
					{
						ContextName: "custom-id",
						Operator:    api.OperatorIn,
						Values:      []string{"custom-ctx"},
					}}},
				{Id: 2, Constraints: []api.Constraint{
					{
						ContextName: "semver",
						Operator:    api.OperatorSemverGt,
						Value:       "3.2.1",
					}}},
				{Id: 4, Constraints: []api.Constraint{
					{
						ContextName: "age",
						Operator:    api.OperatorNumEq,
						Value:       "15",
					}}},
				{Id: 6, Constraints: []api.Constraint{
					{
						ContextName: "domain",
						Operator:    api.OperatorStrStartsWith,
						Values:      []string{"unleash"},
					}}},
			}})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnCount", mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnError").Return()

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	assert.NoError(err)
	client.WaitForReady()

	variant := client.GetVariant(feature, WithVariantContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx", "semver": "3.2.2", "age": "18", "domain": "unleashtest"},
	}))

	assert.False(variant.Enabled)

	assert.False(variant.FeatureEnabled)

	variantFromResolver := client.GetVariant(feature, WithVariantContext(context.Context{
		Properties: map[string]string{"custom-id": "custom-ctx", "semver": "3.2.2", "age": "18", "domain": "unleashtest"},
	}), WithVariantResolver(func(featureName string) *api.Feature {
		if featureName == features[0].Name {
			return &features[0]
		} else {
			t.Fatalf("the feature name passed %s was not the expected one %s", featureName, features[0].Name)
			return nil
		}
	}))

	assert.False(variantFromResolver.Enabled)

	assert.False(variantFromResolver.FeatureEnabled)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClient_ShouldFavorStrategyVariantOverFeatureVariant(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	features := []api.Feature{
		{
			Name:        "feature-x",
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "feature-strategy",
			Strategies: []api.Strategy{
				{
					Id:          1,
					Name:        "default",
					Constraints: []api.Constraint{},
					Parameters: map[string]interface{}{
						"groupId": "strategyVariantName",
					},
					Variants: []api.VariantInternal{
						{
							Variant: api.Variant{
								Name: "strategyVariantName",
								Payload: api.Payload{
									Type:  "string",
									Value: "strategyVariantValue",
								},
							},
							Weight: 1000,
						},
					},
				},
			},
			Variants: []api.VariantInternal{
				{
					Variant: api.Variant{
						Name: "willBeIgnored",
						Payload: api.Payload{
							Type:  "string",
							Value: "willBeIgnored",
						},
						Enabled: true,
					},
					Weight: 100,
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
	mockListener.On("OnCount", mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnError", mock.AnythingOfType("*errors.errorString"))

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	assert.NoError(err)

	client.WaitForReady()

	strategyVariant := client.GetVariant("feature-x")

	assert.True(strategyVariant.Enabled)

	assert.True(strategyVariant.FeatureEnabled)

	assert.Equal("strategyVariantName", strategyVariant.Name)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClient_ShouldReturnOldVariantForNonMatchingStrategyVariant(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	features := []api.Feature{
		{
			Name:        "feature-x",
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "feature-strategy",
			Strategies: []api.Strategy{
				{
					Id:          1,
					Name:        "flexibleRollout",
					Constraints: []api.Constraint{},
					Parameters: map[string]interface{}{
						"rollout":    0,
						"stickiness": "default",
					},
					Variants: []api.VariantInternal{
						{
							Variant: api.Variant{
								Name: "strategyVariantName",
								Payload: api.Payload{
									Type:  "string",
									Value: "strategyVariantValue",
								},
								Enabled: true,
							},
							Weight: 1000,
						},
					},
				},
				{
					Id:          2,
					Name:        "flexibleRollout",
					Constraints: []api.Constraint{},
					Parameters: map[string]interface{}{
						"rollout":    100,
						"stickiness": "default",
					},
				},
			},
			Variants: []api.VariantInternal{
				{
					Variant: api.Variant{
						Name: "willBeSelected",
						Payload: api.Payload{
							Type:  "string",
							Value: "willBeSelected",
						},
						Enabled: true,
					},
					Weight: 100,
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
	mockListener.On("OnCount", mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnError", mock.AnythingOfType("*errors.errorString"))

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	assert.NoError(err)

	client.WaitForReady()

	strategyVariant := client.GetVariant("feature-x")

	assert.True(strategyVariant.Enabled)

	assert.True(strategyVariant.FeatureEnabled)

	assert.Equal("willBeSelected", strategyVariant.Name)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClient_VariantFromEnabledFeatureWithNoVariants(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	feature := "feature-no-variants"
	features := []api.Feature{
		{
			Name:        feature,
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "default-strategy",
			Strategies: []api.Strategy{
				{
					Id:   1,
					Name: "default",
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
	mockListener.On("OnCount", feature, true).Return()
	mockListener.On("OnError").Return()

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(mockListener),
	)

	assert.NoError(err)
	client.WaitForReady()

	variant := client.GetVariant(feature, WithVariantContext(context.Context{}))

	assert.False(variant.Enabled)

	assert.True(variant.FeatureEnabled)

	assert.Equal(disabledVariantFeatureEnabled, variant)

	variantFromResolver := client.GetVariant(feature, WithVariantContext(context.Context{}), WithVariantResolver(func(featureName string) *api.Feature {
		if featureName == features[0].Name {
			return &features[0]
		} else {
			t.Fatalf("the feature name passed %s was not the expected one %s", featureName, features[0].Name)
			return nil
		}
	}))

	assert.False(variantFromResolver.Enabled)

	assert.True(variantFromResolver.FeatureEnabled)

	assert.Equal(disabledVariantFeatureEnabled, variantFromResolver)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestGetVariantWithFallbackVariantWhenFeatureDisabled(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	feature := "feature-disabled"
	features := []api.Feature{
		{
			Name:        feature,
			Description: "feature-desc",
			Enabled:     false,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "default-strategy",
			Strategies: []api.Strategy{
				{
					Id:   1,
					Name: "default",
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

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(&NoopListener{}),
	)

	assert.NoError(err)
	client.WaitForReady()

	fallbackVariant := api.Variant{
		Name: "fallback-variant",
	}

	variant := client.GetVariant(feature, WithVariantFallback(&fallbackVariant))

	assert.False(variant.Enabled)

	assert.False(variant.FeatureEnabled)

	assert.Equal(fallbackVariant, *variant)

	fallbackFunc := func(feature string, ctx *context.Context) *api.Variant {
		return &fallbackVariant
	}

	variantWithFallbackFunc := client.GetVariant(feature, WithVariantFallbackFunc(fallbackFunc))

	assert.Equal(fallbackVariant, *variantWithFallbackFunc)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestGetVariantWithFallbackVariantWhenFeatureEnabledButNoVariants(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	feature := "feature-no-variants"
	features := []api.Feature{
		{
			Name:        feature,
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "default-strategy",
			Strategies: []api.Strategy{
				{
					Id:   1,
					Name: "default",
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

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(&NoopListener{}),
	)

	assert.NoError(err)
	client.WaitForReady()

	fallbackVariant := api.Variant{
		Name: "fallback-variant",
	}

	variant := client.GetVariant(feature, WithVariantFallback(&fallbackVariant))

	assert.False(variant.Enabled)

	// Because we return the fallback variant and the variant
	// doesn't have a FeatureEnabled property, it will always be
	// false, regardless of whether the actual feature is enabled
	// or not.
	assert.False(variant.FeatureEnabled)

	assert.Equal(fallbackVariant, *variant)

	fallbackFunc := func(feature string, ctx *context.Context) *api.Variant {
		return &fallbackVariant
	}

	variantWithFallbackFunc := client.GetVariant(feature, WithVariantFallbackFunc(fallbackFunc))

	assert.Equal(fallbackVariant, *variantWithFallbackFunc)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestGetVariantWithFallbackVariantWhenFeatureDoesntExist(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	feature := "feature-no-variants"
	features := []api.Feature{}

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Features: features,
			Segments: []api.Segment{},
		})

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(&NoopListener{}),
	)

	assert.NoError(err)
	client.WaitForReady()

	fallbackVariant := api.Variant{
		Name: "fallback-variant",
	}

	variant := client.GetVariant(feature, WithVariantFallback(&fallbackVariant))

	assert.False(variant.Enabled)

	assert.False(variant.FeatureEnabled)

	assert.Equal(fallbackVariant, *variant)

	fallbackFunc := func(feature string, ctx *context.Context) *api.Variant {
		return &fallbackVariant
	}

	variantWithFallbackFunc := client.GetVariant(feature, WithVariantFallbackFunc(fallbackFunc))

	assert.Equal(fallbackVariant, *variantWithFallbackFunc)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestGetVariant_FallbackVariantFeatureEnabledSettingIsLeftUnchanged(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-INSTANCEID", mockInstanceId).
		Reply(200)

	enabledFeatureNoVariants := "enabled-feature"
	disabledFeature := "disabled-feature"
	features := []api.Feature{
		{
			Name:        disabledFeature,
			Description: "feature-desc",
			Enabled:     false,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "default-strategy",
			Strategies: []api.Strategy{
				{
					Id:   1,
					Name: "default",
				},
			},
		},
		{
			Name:        enabledFeatureNoVariants,
			Description: "feature-desc",
			Enabled:     true,
			CreatedAt:   time.Date(1974, time.May, 19, 1, 2, 3, 4, time.UTC),
			Strategy:    "default-strategy",
			Strategies: []api.Strategy{
				{
					Id:   1,
					Name: "default",
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

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(&NoopListener{}),
	)

	assert.NoError(err)
	client.WaitForReady()

	fallbackVariantFeatureEnabled := api.Variant{
		Name:           "fallback-variant-feature-enabled",
		FeatureEnabled: true,
	}
	fallbackVariantFeatureDisabled := api.Variant{
		Name:           "fallback-variant-feature-disabled",
		FeatureEnabled: false,
	}

	variantForEnabledFeatureNoVariants := client.GetVariant(enabledFeatureNoVariants, WithVariantFallback(&fallbackVariantFeatureDisabled))

	assert.False(variantForEnabledFeatureNoVariants.FeatureEnabled)

	variantForDisabledFeature := client.GetVariant(disabledFeature, WithVariantFallback(&fallbackVariantFeatureEnabled))

	assert.True(variantForDisabledFeature.FeatureEnabled)

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestSendIdentificationHeaders(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-SDK", `unleash-go-sdk:\d+\.\d+\.\d+`).
		MatchHeader("UNLEASH-CONNECTION-ID", `[0-9a-f\-]{36}`).
		Reply(200)

	gock.New(mockerServer).
		Get("/client/features").
		MatchHeader("UNLEASH-APPNAME", mockAppName).
		MatchHeader("UNLEASH-SDK", `unleash-go-sdk:\d+\.\d+\.\d+`).
		MatchHeader("UNLEASH-CONNECTION-ID", `[0-9a-f\-]{36}`).
		Reply(200).
		JSON(api.FeatureResponse{})

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(&NoopListener{}),
	)

	assert.NoError(err)

	client.WaitForReady()

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestConnectionAndIntervalHeadersAndBody(t *testing.T) {
	assert := assert.New(t)
	defer gock.OffAll()

	gock.Observe(gock.DumpRequest)

	gock.New(mockerServer).
		Post("/client/register").
		MatchHeader("UNLEASH-CONNECTION-ID", `[0-9a-f\-]{36}`).
		BodyString(`.*connectionId.*`).
		Reply(200)

	gock.New(mockerServer).
		Post("/client/metrics").
		MatchHeader("UNLEASH-CONNECTION-ID", `[0-9a-f\-]{36}`).
		MatchHeader("Unleash-Interval", "50").
		BodyString(`.*connectionId.*`).
		Reply(200)

	gock.New(mockerServer).
		Get("/client/features").
		MatchHeader("UNLEASH-CONNECTION-ID", `[0-9a-f\-]{36}`).
		MatchHeader("Unleash-Interval", "100").
		Reply(200).
		JSON(api.FeatureResponse{})

	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithMetricsInterval(50*time.Millisecond),
		WithRefreshInterval(100*time.Millisecond),
		WithInstanceId(mockInstanceId),
		WithListener(&NoopListener{}),
		WithDisableMetrics(false),
	)

	assert.NoError(err)

	client.IsEnabled("foo")

	time.Sleep(100 * time.Millisecond)
	err = client.Close()

	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestClient_DisablePollingDoesNotFetchFeatures(t *testing.T) {
	assert := assert.New(t)
	var featuresCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method + " " + req.URL.Path {
		case "GET /client/features":
			atomic.AddInt32(&featuresCalls, 1)
			rw.WriteHeader(http.StatusOK)
			writeJSON(rw, api.FeatureResponse{})
		default:
			t.Fatalf("Unexpected request: %+v", req)
		}
	}))
	defer srv.Close()

	client, err := NewClient(
		WithUrl(srv.URL),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithDisableMetrics(true),
		WithDisablePolling(true),
		WithRefreshInterval(time.Millisecond),
	)
	assert.Nil(err, "client should not return an error")

	time.Sleep(25 * time.Millisecond)
	assert.Equal(int32(0), atomic.LoadInt32(&featuresCalls))
	assert.Nil(client.Close())
}

func TestClient_DisablePollingUsesBootstrappedFeatures(t *testing.T) {
	assert := assert.New(t)
	var featuresCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" && req.URL.Path == "/client/features" {
			atomic.AddInt32(&featuresCalls, 1)
		}
		t.Fatalf("Unexpected request: %+v", req)
	}))
	defer srv.Close()

	bootstrap := bytes.NewBufferString(`{
		"version": 2,
		"features": [
			{
				"name": "bootstrapped-feature",
				"enabled": true,
				"strategies": [
					{
						"name": "default"
					}
				],
				"variants": []
			}
		]
	}`)

	client, err := NewClient(
		WithUrl(srv.URL),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithDisableMetrics(true),
		WithDisablePolling(true),
		WithStorage(&BootstrapStorage{Reader: bootstrap}),
		WithRefreshInterval(time.Millisecond),
	)
	assert.Nil(err, "client should not return an error")

	assert.True(client.IsEnabled("bootstrapped-feature"))
	time.Sleep(25 * time.Millisecond)
	assert.Equal(int32(0), atomic.LoadInt32(&featuresCalls))
	assert.Nil(client.Close())
}

func TestClient_DisablePollingAllowsSynchronousInitialFetch(t *testing.T) {
	assert := assert.New(t)
	var featuresCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method + " " + req.URL.Path {
		case "GET /client/features":
			atomic.AddInt32(&featuresCalls, 1)
			rw.WriteHeader(http.StatusOK)
			writeJSON(rw, api.FeatureResponse{
				Features: []api.Feature{
					{
						Name:    "synchronously-fetched-feature",
						Enabled: true,
						Strategies: []api.Strategy{
							{
								Name: "default",
							},
						},
					},
				},
			})
		default:
			t.Fatalf("Unexpected request: %+v", req)
		}
	}))
	defer srv.Close()

	client, err := NewClient(
		WithUrl(srv.URL),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithDisableMetrics(true),
		WithDisablePolling(true),
		WithSynchronousFetchOnInitialisation(true),
		WithRefreshInterval(time.Millisecond),
	)
	assert.Nil(err, "client should not return an error")

	assert.True(client.IsEnabled("synchronously-fetched-feature"))
	time.Sleep(25 * time.Millisecond)
	assert.Equal(int32(1), atomic.LoadInt32(&featuresCalls))
	assert.Nil(client.Close())
}

func TestClient_DisablePollingReadyFiresImmediately(t *testing.T) {
	assert := assert.New(t)

	client, err := NewClient(
		WithUrl("http://localhost:1"),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithDisableMetrics(true),
		WithDisablePolling(true),
	)
	assert.Nil(err, "client should not return an error")

	done := make(chan struct{})
	go func() {
		client.WaitForReady()
		close(done)
	}()
	select {
	case <-done:
	case <-time.NewTimer(500 * time.Millisecond).C:
		t.Fatal("WaitForReady() did not return immediately in static mode")
	}
	assert.Nil(client.Close())
}

func TestClient_DisablePollingCloseDoesNotDeadlock(t *testing.T) {
	assert := assert.New(t)

	client, err := NewClient(
		WithUrl("http://localhost:1"),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithDisableMetrics(true),
		WithDisablePolling(true),
	)
	assert.Nil(err, "client should not return an error")

	done := make(chan struct{})
	go func() {
		client.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.NewTimer(2 * time.Second).C:
		t.Fatal("Close() deadlocked in static mode")
	}
}

func TestClient_SynchronousFetchWithPollingContinues(t *testing.T) {
	assert := assert.New(t)
	var featuresCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.Method + " " + req.URL.Path {
		case "GET /client/features":
			atomic.AddInt32(&featuresCalls, 1)
			rw.WriteHeader(http.StatusOK)
			writeJSON(rw, api.FeatureResponse{
				Features: []api.Feature{
					{
						Name:    "sync-polled-feature",
						Enabled: true,
						Strategies: []api.Strategy{
							{Name: "default"},
						},
					},
				},
			})
		default:
			t.Fatalf("Unexpected request: %+v", req)
		}
	}))
	defer srv.Close()

	client, err := NewClient(
		WithUrl(srv.URL),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithDisableMetrics(true),
		WithSynchronousFetchOnInitialisation(true),
		WithRefreshInterval(10*time.Millisecond),
	)
	assert.Nil(err, "client should not return an error")

	assert.True(client.IsEnabled("sync-polled-feature"))
	assert.GreaterOrEqual(atomic.LoadInt32(&featuresCalls), int32(1), "expected at least one fetch on init")

	time.Sleep(50 * time.Millisecond)
	assert.Greater(atomic.LoadInt32(&featuresCalls), int32(1), "expected continued polling after sync fetch")

	assert.Nil(client.Close())
}
