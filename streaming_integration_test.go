package unleash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamingModeConfiguration(t *testing.T) {
	// Test that streaming mode can be configured
	client, err := NewClient(
		WithUrl("https://example.com"),
		WithAppName("test-app"),
		WithExperimentalMode(map[string]string{
			"type": "streaming",
		}),
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	
	// Check that streaming mode is enabled
	assert.True(t, client.options.IsStreamingMode())
	
	// Check that repository was configured for streaming
	assert.True(t, client.repository.isStreaming)
	
	// Cleanup
	client.Close()
}

func TestNonStreamingModeByDefault(t *testing.T) {
	// Test that non-streaming mode is default
	client, err := NewClient(
		WithUrl("https://example.com"),
		WithAppName("test-app"),
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	
	// Check that streaming mode is disabled by default
	assert.False(t, client.options.IsStreamingMode())
	
	// Check that repository was configured for polling
	assert.False(t, client.repository.isStreaming)
	
	// Cleanup
	client.Close()
}

func TestExperimentalModeHelpers(t *testing.T) {
	// Test streaming mode detection
	config := &configOption{
		experimentalMode: map[string]string{
			"type": "streaming",
		},
	}
	assert.True(t, config.IsStreamingMode())

	// Test non-streaming mode
	config2 := &configOption{
		experimentalMode: map[string]string{
			"type": "polling",
		},
	}
	assert.False(t, config2.IsStreamingMode())

	// Test no experimental mode
	config3 := &configOption{}
	assert.False(t, config3.IsStreamingMode())
}