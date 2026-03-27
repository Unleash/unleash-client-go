package unleash

import (
	"github.com/Unleash/unleash-go-sdk/v6/api"
)

type mockEvent struct {
	id    string
	event string
	data  string
}

func (m *mockEvent) Id() string    { return m.id }
func (m *mockEvent) Event() string { return m.event }
func (m *mockEvent) Data() string  { return m.data }

type NoOpStorage struct{}

func (s *NoOpStorage) Persist(features *api.FeatureResponse) error {
	return nil
}

func (s *NoOpStorage) Load() (*api.FeatureResponse, error) {
	return &api.FeatureResponse{}, nil
}

func (s *NoOpStorage) Init(backupPath, appName string) {}

// func TestStreamingClient_HandleEvents(t *testing.T) {
// 	errChannels := errorChannels{
// 		errors:   make(chan error, 10),
// 		warnings: make(chan error, 10),
// 	}
// 	repoChannels := repositoryChannels{
// 		errorChannels: errChannels,
// 		ready:         make(chan bool, 1),
// 		update:        make(chan bool, 1),
// 	}

// 	options := repositoryOptions{
// 		url:        url.URL{},
// 		appName:    "test-app",
// 		instanceId: "test-instance",
// 		httpClient: &http.Client{},
// 		headers:    make(http.Header),
// 		storage:    &NoOpStorage{},
// 	}

// 	client := newStreamingClient(options, repoChannels)
// 	client.ctx, client.cancel = context.WithCancel(context.Background())

// 	connectedData := map[string]interface{}{
// 		"events": []map[string]interface{}{
// 			{
// 				"type":    "hydration",
// 				"eventId": 1,
// 				"features": []map[string]interface{}{
// 					{
// 						"name":    "test-feature",
// 						"enabled": true,
// 					},
// 				},
// 				"segments": []interface{}{},
// 			},
// 		},
// 	}
// 	connectedJSON, _ := json.Marshal(connectedData)

// 	err := client.handleDomainEvent(&mockEvent{
// 		event: "unleash-connected",
// 		data:  string(connectedJSON),
// 	})
// 	assert.NoError(t, err)

// 	snapshot := client.snapshot()

// 	feature, found := snapshot.Features["test-feature"]
// 	assert.True(t, found)
// 	assert.True(t, feature.Enabled)

// 	updatedData := map[string]interface{}{
// 		"events": []map[string]interface{}{
// 			{
// 				"type":    "feature-updated",
// 				"eventId": 2,
// 				"feature": map[string]interface{}{
// 					"name":    "test-feature",
// 					"enabled": false,
// 				},
// 			},
// 		},
// 	}
// 	updatedJSON, _ := json.Marshal(updatedData)

// 	err = client.handleDomainEvent(&mockEvent{
// 		event: "unleash-updated",
// 		data:  string(updatedJSON),
// 	})
// 	assert.NoError(t, err)

// 	snapshot = client.snapshot()

// 	feature, found = snapshot.Features["test-feature"]
// 	assert.True(t, found)
// 	assert.False(t, feature.Enabled)

// 	client.cancel()
// }
