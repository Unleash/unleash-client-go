package unleash

import (
	"sync"
	"testing"

	"github.com/Unleash/unleash-go-sdk/v5/api"
	"github.com/Unleash/unleash-go-sdk/v5/context"
	"github.com/h2non/gock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupClient(t *testing.T, listener *MockedListener) *Client {
	client, err := NewClient(
		WithUrl(mockerServer),
		WithAppName(mockAppName),
		WithInstanceId(mockInstanceId),
		WithListener(listener),
	)
	assert.NoError(t, err)
	client.WaitForReady()
	return client
}

func TestImpression_Off(t *testing.T) {
	defer gock.OffAll()
	assert := assert.New(t)

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	feature := "impression-data-off"

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Features: []api.Feature{
				{
					Name:    feature,
					Enabled: true,
					Strategies: []api.Strategy{
						{
							Id:   1,
							Name: "flexibleRollout",
							Parameters: map[string]any{
								"rollout":    100,
								"stickiness": "default",
							},
						},
					},
				},
			},
		})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, true).Return()
	mockListener.On("OnImpression", mock.Anything).Maybe()

	client := setupClient(t, mockListener)
	client.IsEnabled(feature)

	assert.NoError(client.Close())
	mockListener.AssertNotCalled(t, "OnImpression", mock.Anything)
	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestImpression_IsEnabled(t *testing.T) {
	defer gock.OffAll()
	assert := assert.New(t)
	var wg sync.WaitGroup
	wg.Add(1)

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	feature := "impression-data-on"

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Features: []api.Feature{
				{
					Name:           feature,
					Enabled:        true,
					ImpressionData: true,
					Strategies: []api.Strategy{
						{
							Id:   1,
							Name: "flexibleRollout",
							Parameters: map[string]any{
								"rollout":    100,
								"stickiness": "default",
							},
						},
					},
				},
			},
		})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, true).Return()

	mockListener.On("OnImpression", mock.MatchedBy(func(e ImpressionEvent) bool {
		return e.FeatureName == feature &&
			e.EventType == ImpressionEventTypeIsEnabled &&
			e.Enabled == true
	})).Run(func(args mock.Arguments) {
		wg.Done()
	}).Once()

	client := setupClient(t, mockListener)
	client.IsEnabled(feature)

	wg.Wait()
	assert.NoError(client.Close())
	mockListener.AssertExpectations(t)
	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestImpression_GetVariant(t *testing.T) {
	defer gock.OffAll()
	assert := assert.New(t)
	var wg sync.WaitGroup
	wg.Add(1)

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	feature := "impression-data-on-variant"
	variant := "variant-for-impression-data"

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Features: []api.Feature{
				{
					Name:           feature,
					Enabled:        true,
					ImpressionData: true,
					Strategies: []api.Strategy{
						{
							Id:   1,
							Name: "flexibleRollout",
							Parameters: map[string]any{
								"rollout":    100,
								"stickiness": "default",
								"groupId":    variant,
							},
							Variants: []api.VariantInternal{
								{
									Variant: api.Variant{
										Name: variant,
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
				},
			},
		})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, true).Return()

	mockListener.On("OnImpression", mock.MatchedBy(func(e ImpressionEvent) bool {
		return e.FeatureName == feature &&
			e.EventType == ImpressionEventTypeGetVariant &&
			e.Enabled == true &&
			e.Variant == variant
	})).Run(func(args mock.Arguments) {
		wg.Done()
	}).Once()

	client := setupClient(t, mockListener)
	client.GetVariant(feature)

	wg.Wait()
	assert.NoError(client.Close())
	mockListener.AssertExpectations(t)
	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestImpression_WithContext(t *testing.T) {
	defer gock.OffAll()
	assert := assert.New(t)
	var wg sync.WaitGroup
	wg.Add(1)

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	feature := "with-context"
	ctxUserId := "123"
	ctxSessionId := "abc"
	ctxPropertyId := "impression-data-ctx-id"
	ctxPropertyValue := "impression-data-ctx-value"

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Features: []api.Feature{
				{
					Name:           feature,
					Enabled:        true,
					ImpressionData: true,
					Strategies: []api.Strategy{
						{
							Id:   1,
							Name: "flexibleRollout",
							Parameters: map[string]any{
								"rollout":    100,
								"stickiness": "default",
							},
							Constraints: []api.Constraint{
								{
									ContextName: ctxPropertyId,
									Operator:    api.OperatorIn,
									Values:      []string{ctxPropertyValue},
								},
							},
						},
					},
				},
			},
		})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, true).Return()

	userCtx := context.Context{
		UserId:    ctxUserId,
		SessionId: ctxSessionId,
		Properties: map[string]string{
			ctxPropertyId: ctxPropertyValue,
		},
	}

	mockListener.On("OnImpression", mock.MatchedBy(func(e ImpressionEvent) bool {
		return e.FeatureName == feature &&
			e.EventType == ImpressionEventTypeIsEnabled &&
			e.Enabled == true &&
			e.Context != nil &&
			e.Context.UserId == ctxUserId &&
			e.Context.SessionId == ctxSessionId &&
			e.Context.Properties[ctxPropertyId] == ctxPropertyValue
	})).Run(func(args mock.Arguments) {
		wg.Done()
	}).Once()

	client := setupClient(t, mockListener)
	client.IsEnabled(feature, WithContext(userCtx))

	wg.Wait()
	assert.NoError(client.Close())
	mockListener.AssertExpectations(t)
	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestImpression_WithContextAndMultipleEvents(t *testing.T) {
	defer gock.OffAll()
	assert := assert.New(t)
	var wg sync.WaitGroup
	wg.Add(2)

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	feature := "context-multiple-evals"
	ctxUserId := "123"
	ctxSessionId := "abc"
	ctxPropertyId := "ctx-key"
	ctxPropertyValue := "ctx-val"

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Features: []api.Feature{
				{
					Name:           feature,
					Enabled:        true,
					ImpressionData: true,
					Strategies: []api.Strategy{
						{
							Id:   1,
							Name: "flexibleRollout",
							Parameters: map[string]any{
								"rollout":    100,
								"stickiness": "default",
							},
							Constraints: []api.Constraint{
								{
									ContextName: ctxPropertyId,
									Operator:    api.OperatorIn,
									Values:      []string{ctxPropertyValue},
								},
							},
						},
					},
				},
			},
		})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, true).Maybe()
	mockListener.On("OnCount", feature, false).Maybe()

	userCtx := context.Context{
		UserId:    ctxUserId,
		SessionId: ctxSessionId,
		Properties: map[string]string{
			ctxPropertyId: ctxPropertyValue,
		},
	}

	mockListener.On("OnImpression", mock.MatchedBy(func(e ImpressionEvent) bool {
		return e.FeatureName == feature &&
			e.EventType == ImpressionEventTypeIsEnabled &&
			e.Enabled == true &&
			e.Context != nil &&
			e.Context.UserId == ctxUserId &&
			e.Context.SessionId == ctxSessionId &&
			e.Context.Properties[ctxPropertyId] == ctxPropertyValue
	})).Run(func(args mock.Arguments) {
		wg.Done()
	}).Once()

	mockListener.On("OnImpression", mock.MatchedBy(func(e ImpressionEvent) bool {
		return e.FeatureName == feature &&
			e.EventType == ImpressionEventTypeIsEnabled &&
			e.Enabled == false &&
			len(e.Context.Properties) == 0
	})).Run(func(args mock.Arguments) {
		wg.Done()
	}).Once()

	client := setupClient(t, mockListener)

	resultWithCtx := client.IsEnabled(feature, WithContext(userCtx))
	assert.True(resultWithCtx)

	resultWithoutCtx := client.IsEnabled(feature)
	assert.False(resultWithoutCtx)

	wg.Wait()
	assert.NoError(client.Close())
	mockListener.AssertExpectations(t)
	assert.True(gock.IsDone(), "there should be no more mocks")
}

func TestImpression_GetChannelMethod(t *testing.T) {
	defer gock.OffAll()
	assert := assert.New(t)
	var wg sync.WaitGroup
	wg.Add(1)

	gock.New(mockerServer).
		Post("/client/register").
		Reply(200)

	feature := "impression-get-channel-method-test"

	gock.New(mockerServer).
		Get("/client/features").
		Reply(200).
		JSON(api.FeatureResponse{
			Features: []api.Feature{
				{
					Name:           feature,
					Enabled:        true,
					ImpressionData: true,
					Strategies: []api.Strategy{
						{
							Id:   1,
							Name: "flexibleRollout",
							Parameters: map[string]any{
								"rollout":    100,
								"stickiness": "default",
							},
						},
					},
				},
			},
		})

	mockListener := &MockedListener{}
	mockListener.On("OnReady").Return()
	mockListener.On("OnRegistered", mock.AnythingOfType("ClientData"))
	mockListener.On("OnCount", feature, true).Return()
	mockListener.On("OnImpression", mock.AnythingOfType("ImpressionEvent")).Maybe()
	mockListener.On("OnError", mock.Anything).Maybe()

	client := setupClient(t, mockListener)
	impressionChannel := client.Impression()

	ready := make(chan struct{})

	go func() {
		close(ready)
		for impression := range impressionChannel {
			if impression.FeatureName == feature &&
				impression.EventType == ImpressionEventTypeIsEnabled &&
				impression.Enabled {
				wg.Done()
				return
			}
		}
	}()

	<-ready
	client.IsEnabled(feature)

	wg.Wait()
	assert.NoError(client.Close())
	mockListener.AssertExpectations(t)
	assert.True(gock.IsDone(), "there should be no more mocks")
}
