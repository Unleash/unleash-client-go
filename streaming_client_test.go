package unleash

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5/api"
	"github.com/r3labs/sse/v2"
	"github.com/stretchr/testify/assert"
)

func TestStreamingClient_Creation(t *testing.T) {
	// Parse server URL
	serverURL, _ := url.Parse("http://localhost:8080/")

	// Create channels
	errChannels := errorChannels{
		errors:   make(chan error, 10),
		warnings: make(chan error, 10),
	}
	repoChannels := repositoryChannels{
		errorChannels: errChannels,
		ready:         make(chan bool, 1),
		update:        make(chan bool, 1),
	}

	// Create repository options
	options := repositoryOptions{
		url:        *serverURL,
		appName:    "test-app",
		instanceId: "test-instance",
		httpClient: &http.Client{Timeout: 5 * time.Second},
		headers:    make(http.Header),
	}

	// Create mock repository
	repo := &repository{
		segments: make(map[int][]api.Constraint),
	}

	// Create streaming client
	client := newStreamingClient(options, repo, repoChannels, errChannels)

	// Verify client was created correctly
	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8080/client/streaming", client.url)
	assert.Equal(t, "test-app", client.appName)
	assert.Equal(t, "test-instance", client.instanceId)
	assert.False(t, client.isRunning())
}

func TestStreamingProcessor_ProcessFeatureResponse(t *testing.T) {
	// Create channels
	errChannels := errorChannels{
		errors:   make(chan error, 10),
		warnings: make(chan error, 10),
	}
	repoChannels := repositoryChannels{
		errorChannels: errChannels,
		ready:         make(chan bool, 1),
		update:        make(chan bool, 1),
	}

	// Create storage
	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")

	// Create mock repository
	repo := &repository{
		segments: make(map[int][]api.Constraint),
	}

	// Create processor
	processor := newStreamingProcessor(storage, repo, repoChannels)

	// Create feature response
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

	// Process the response
	err := processor.processFeatureResponse(response)
	assert.NoError(t, err)

	// Check if features were stored
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

	// Check segments - they are stored in the repository
	// Segments are handled directly by the repository for both polling and streaming modes

	// Check that ready signal was sent
	select {
	case <-repoChannels.ready:
		// Expected
	default:
		t.Error("Expected ready signal to be sent")
	}
}

func TestStreamingClient_HandleEvents(t *testing.T) {
	// Create channels
	errChannels := errorChannels{
		errors:   make(chan error, 10),
		warnings: make(chan error, 10),
	}
	repoChannels := repositoryChannels{
		errorChannels: errChannels,
		ready:         make(chan bool, 1),
		update:        make(chan bool, 1),
	}

	// Create repository options
	options := repositoryOptions{
		url:        url.URL{},
		appName:    "test-app",
		instanceId: "test-instance",
		httpClient: &http.Client{},
		headers:    make(http.Header),
	}

	// Create mock repository
	repo := &repository{
		segments: make(map[int][]api.Constraint),
	}

	// Create streaming client
	client := newStreamingClient(options, repo, repoChannels, errChannels)
	client.ctx, client.cancel = context.WithCancel(context.Background())

	// Create storage and processor
	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")
	client.processor = newStreamingProcessor(storage, repo, repoChannels)

	// Test connected event
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

	err := client.handleConnectedEvent(&sse.Event{
		Event: []byte("unleash-connected"),
		Data:  connectedJSON,
	})
	assert.NoError(t, err)

	// Check if feature was processed
	feature, found := storage.Get("test-feature")
	assert.True(t, found)
	if f, ok := feature.(api.Feature); ok {
		assert.True(t, f.Enabled)
	}

	// Test updated event
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

	err = client.handleUpdatedEvent(&sse.Event{
		Event: []byte("unleash-updated"),
		Data:  updatedJSON,
	})
	assert.NoError(t, err)

	// Check if feature was updated
	feature, found = storage.Get("test-feature")
	assert.True(t, found)
	if f, ok := feature.(api.Feature); ok {
		assert.False(t, f.Enabled)
	}

	client.cancel()
}