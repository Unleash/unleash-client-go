package unleash

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5/api"
	"github.com/stretchr/testify/assert"
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

func TestStreamingClient_Creation(t *testing.T) {
	serverURL, _ := url.Parse("http://localhost:8080/")

	errChannels := errorChannels{
		errors:   make(chan error, 10),
		warnings: make(chan error, 10),
	}
	repoChannels := repositoryChannels{
		errorChannels: errChannels,
		ready:         make(chan bool, 1),
		update:        make(chan bool, 1),
	}

	options := repositoryOptions{
		url:        *serverURL,
		appName:    "test-app",
		instanceId: "test-instance",
		httpClient: &http.Client{Timeout: 5 * time.Second},
		headers:    make(http.Header),
	}

	repo := &repository{}

	// Create a delta processor for the test
	deltaProc := newDeltaProcessor(repo, repoChannels)

	client := newStreamingClient(options, repoChannels, deltaProc)

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8080/client/streaming", client.url)
	assert.Equal(t, "test-app", client.appName)
	assert.Equal(t, "test-instance", client.instanceId)
	assert.False(t, client.isRunning())
}

func TestStreamingClient_HandleEvents(t *testing.T) {
	errChannels := errorChannels{
		errors:   make(chan error, 10),
		warnings: make(chan error, 10),
	}
	repoChannels := repositoryChannels{
		errorChannels: errChannels,
		ready:         make(chan bool, 1),
		update:        make(chan bool, 1),
	}

	options := repositoryOptions{
		url:        url.URL{},
		appName:    "test-app",
		instanceId: "test-instance",
		httpClient: &http.Client{},
		headers:    make(http.Header),
		storage:    &NoOpStorage{},
	}

	repo := &repository{
		options: options,
	}

	// Create a delta processor for the test
	deltaProc := newDeltaProcessor(repo, repoChannels)

	client := newStreamingClient(options, repoChannels, deltaProc)
	client.ctx, client.cancel = context.WithCancel(context.Background())

	connectedData := map[string]interface{}{
		"events": []map[string]interface{}{
			{
				"type":    "hydration",
				"eventId": 1,
				"features": []map[string]interface{}{
					{
						"name":    "test-feature",
						"enabled": true,
					},
				},
				"segments": []interface{}{},
			},
		},
	}
	connectedJSON, _ := json.Marshal(connectedData)

	err := client.handleConnectedEvent(&mockEvent{
		event: "unleash-connected",
		data:  string(connectedJSON),
	})
	assert.NoError(t, err)

	snapshot := repo.snapshot()

	feature, found := snapshot.Features["test-feature"]
	assert.True(t, found)
	assert.True(t, feature.Enabled)

	updatedData := map[string]interface{}{
		"events": []map[string]interface{}{
			{
				"type":    "feature-updated",
				"eventId": 2,
				"feature": map[string]interface{}{
					"name":    "test-feature",
					"enabled": false,
				},
			},
		},
	}
	updatedJSON, _ := json.Marshal(updatedData)

	err = client.handleUpdatedEvent(&mockEvent{
		event: "unleash-updated",
		data:  string(updatedJSON),
	})
	assert.NoError(t, err)

	snapshot = repo.snapshot()

	feature, found = snapshot.Features["test-feature"]
	assert.True(t, found)
	assert.False(t, feature.Enabled)

	client.cancel()
}
