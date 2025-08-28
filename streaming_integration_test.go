package unleash

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStreamingModeConfiguration(t *testing.T) {
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
	
	client.Close()
}

func TestNonStreamingModeByDefault(t *testing.T) {
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