package unleash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
