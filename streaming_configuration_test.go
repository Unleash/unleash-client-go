package unleash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStreamingModeConfiguration(t *testing.T) {
	// Use a mock server instead of external URL
	server := mockSSEServer(`{"events":[{"type":"hydration","eventId":1,"features":[],"segments":[]}]}`, "")
	defer server.Close()

	listener := &MockedListener{}
	listener.On("OnError", mock.Anything).Return()
	listener.On("OnWarning", mock.Anything).Return()
	listener.On("OnReady").Return()
	listener.On("OnRegistered", mock.AnythingOfType("ClientData")).Return()
	listener.On("OnCount", mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return()

	client, err := NewClient(
		WithUrl(server.URL),
		WithAppName("test-app"),
		WithListener(listener),
		WithExperimentalMode(map[string]string{
			"type": "streaming",
		}),
		WithDisableMetrics(true),
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)

	assert.True(t, client.options.IsStreamingMode())
	if _, ok := client.repository.(*repository); !ok {
		t.Fatalf("expected streamingRepository, got %T", client.repository)
	}

	client.WaitForReady()
	client.Close()
}

func TestNonStreamingModeByDefault(t *testing.T) {
	// Simply test that the default configuration doesn't use streaming mode
	// without trying to connect to any server
	config := &configOption{}

	// Check that streaming mode is disabled by default
	assert.False(t, config.IsStreamingMode())

	// Test with a client that has an empty experimental mode
	config2 := &configOption{
		experimentalMode: map[string]string{},
	}
	assert.False(t, config2.IsStreamingMode())
}

func TestExperimentalModeHelpers(t *testing.T) {
	config := &configOption{
		experimentalMode: map[string]string{
			"type": "streaming",
		},
	}
	assert.True(t, config.IsStreamingMode())

	config2 := &configOption{
		experimentalMode: map[string]string{
			"type": "polling",
		},
	}
	assert.False(t, config2.IsStreamingMode())

	config3 := &configOption{}
	assert.False(t, config3.IsStreamingMode())
}
