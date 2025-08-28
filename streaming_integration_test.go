package unleash

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStreamingModeConfiguration(t *testing.T) {
	// Set short retry duration for test
	originalEnv := os.Getenv("UNLEASH_STREAMING_RETRY_DURATION_SECONDS")
	os.Setenv("UNLEASH_STREAMING_RETRY_DURATION_SECONDS", "3")
	defer func() {
		if originalEnv != "" {
			os.Setenv("UNLEASH_STREAMING_RETRY_DURATION_SECONDS", originalEnv)
		} else {
			os.Unsetenv("UNLEASH_STREAMING_RETRY_DURATION_SECONDS")
		}
	}()

	listener := &MockedListener{}
	listener.On("OnError", mock.Anything).Return()
	listener.On("OnWarning", mock.Anything).Return()
	listener.On("OnReady").Return()
	listener.On("OnRegistered", mock.AnythingOfType("ClientData")).Return()
	listener.On("OnCount", mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return()

	client, err := NewClient(
		WithUrl("https://example.com"),
		WithAppName("test-app"),
		WithListener(listener),
		WithExperimentalMode(map[string]string{
			"type": "streaming",
		}),
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	
	assert.True(t, client.options.IsStreamingMode())
	assert.True(t, client.repository.IsStreaming())
	
	// Wait for retries to complete (should take ~3 seconds now with env var)
	time.Sleep(4 * time.Second)
	
	client.Close()
}

func TestNonStreamingModeByDefault(t *testing.T) {
	// Set short retry duration for test (though this test shouldn't use streaming)
	originalEnv := os.Getenv("UNLEASH_STREAMING_RETRY_DURATION_SECONDS")
	os.Setenv("UNLEASH_STREAMING_RETRY_DURATION_SECONDS", "3")
	defer func() {
		if originalEnv != "" {
			os.Setenv("UNLEASH_STREAMING_RETRY_DURATION_SECONDS", originalEnv)
		} else {
			os.Unsetenv("UNLEASH_STREAMING_RETRY_DURATION_SECONDS")
		}
	}()

	listener := &MockedListener{}
	listener.On("OnError", mock.Anything).Return()
	listener.On("OnWarning", mock.Anything).Return()
	listener.On("OnReady").Return()
	listener.On("OnRegistered", mock.AnythingOfType("ClientData")).Return()
	listener.On("OnCount", mock.AnythingOfType("string"), mock.AnythingOfType("bool")).Return()

	client, err := NewClient(
		WithUrl("https://example.com"),
		WithAppName("test-app"),
		WithListener(listener),
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	
	assert.False(t, client.options.IsStreamingMode())
	assert.False(t, client.repository.IsStreaming())
	
	client.Close()
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