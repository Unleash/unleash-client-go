package unleash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamingModeConfiguration(t *testing.T) {
	client, err := NewClient(
		WithUrl("https://example.com"),
		WithAppName("test-app"),
		WithExperimentalMode(map[string]string{
			"type": "streaming",
		}),
	)

	assert.NoError(t, err)
	assert.NotNil(t, client)
	
	assert.True(t, client.options.IsStreamingMode())
	assert.True(t, client.repository.IsStreaming())
	
	client.Close()
}

func TestNonStreamingModeByDefault(t *testing.T) {
	client, err := NewClient(
		WithUrl("https://example.com"),
		WithAppName("test-app"),
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