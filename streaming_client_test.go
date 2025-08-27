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

// mockEvent implements the eventsource.Event interface for testing
type mockEvent struct {
	id    string
	event string
	data  string
}

func (m *mockEvent) Id() string    { return m.id }
func (m *mockEvent) Event() string { return m.event }
func (m *mockEvent) Data() string  { return m.data }

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

	repo := &repository{
		segments: make(map[int][]api.Constraint),
	}

	client := newStreamingClient(options, repo, repoChannels, errChannels)

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8080/client/streaming", client.url)
	assert.Equal(t, "test-app", client.appName)
	assert.Equal(t, "test-instance", client.instanceId)
	assert.False(t, client.isRunning())
}

func TestStreamingProcessor_ProcessFeatureResponse(t *testing.T) {
	errChannels := errorChannels{
		errors:   make(chan error, 10),
		warnings: make(chan error, 10),
	}
	repoChannels := repositoryChannels{
		errorChannels: errChannels,
		ready:         make(chan bool, 1),
		update:        make(chan bool, 1),
	}

	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")

	processor := newStreamingProcessor(storage, repoChannels)

	response := api.FeatureResponse{
		Features: []api.Feature{
			{
				Name:    "feature1",
				Enabled: true,
			},
			{
				Name:    "feature2",
				Enabled: false,
			},
		},
		Segments: []api.Segment{
			{
				Id: 1,
				Constraints: []api.Constraint{
					{
						ContextName: "userId",
						Operator:    "IN",
						Values:      []string{"user1", "user2"},
					},
				},
			},
		},
	}

	err := processor.processFeatureResponse(response)
	assert.NoError(t, err)

	feature1, found := storage.Get("feature1")
	assert.True(t, found)
	if f, ok := feature1.(api.Feature); ok {
		assert.True(t, f.Enabled)
	}

	feature2, found := storage.Get("feature2")
	assert.True(t, found)
	if f, ok := feature2.(api.Feature); ok {
		assert.False(t, f.Enabled)
	}

	select {
	case <-repoChannels.ready:
	default:
		t.Error("Expected ready signal to be sent")
	}
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
	}

	repo := &repository{
		segments: make(map[int][]api.Constraint),
	}

	client := newStreamingClient(options, repo, repoChannels, errChannels)
	client.ctx, client.cancel = context.WithCancel(context.Background())

	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")
	client.processor = newStreamingProcessor(storage, repoChannels)

	connectedData := map[string]interface{}{
		"features": []map[string]interface{}{
			{
				"name":    "test-feature",
				"enabled": true,
			},
		},
		"segments": []interface{}{},
	}
	connectedJSON, _ := json.Marshal(connectedData)

	err := client.handleConnectedEvent(&mockEvent{
		event: "unleash-connected",
		data:  string(connectedJSON),
	})
	assert.NoError(t, err)

	feature, found := storage.Get("test-feature")
	assert.True(t, found)
	if f, ok := feature.(api.Feature); ok {
		assert.True(t, f.Enabled)
	}

	updatedData := map[string]interface{}{
		"features": []map[string]interface{}{
			{
				"name":    "test-feature",
				"enabled": false,
			},
		},
		"segments": []interface{}{},
	}
	updatedJSON, _ := json.Marshal(updatedData)

	err = client.handleUpdatedEvent(&mockEvent{
		event: "unleash-updated",
		data:  string(updatedJSON),
	})
	assert.NoError(t, err)

	feature, found = storage.Get("test-feature")
	assert.True(t, found)
	if f, ok := feature.(api.Feature); ok {
		assert.False(t, f.Enabled)
	}

	client.cancel()
}