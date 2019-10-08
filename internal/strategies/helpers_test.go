package strategies

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestResolveHostname(t *testing.T) {
	envHostname := os.Getenv("HOSTNAME")
	osHostname, err := os.Hostname()

	assert.Nil(t, err, "os.Hostname should not return an error")

	hostname, err := resolveHostname()

	assert.Nil(t, err, "resolveHostname should not return an error")

	if envHostname != "" {
		assert.Equal(t, envHostname, hostname)
	} else {
		assert.Equal(t, osHostname, hostname)
	}
}

func TestParameterAsFloat64(t *testing.T) {
	goodData := map[interface{}]float64{
		"30":    30.0,
		"-0.01": -0.01,
		42:      42.0,
		0.1:     0.1,
	}

	for input, expected := range goodData {
		actual, ok := parameterAsFloat64(input)
		if !ok {
			t.Errorf("Not OK: %v", input)
		}
		assert.InDelta(t, actual, expected, 0.0000001)
	}

	badData := map[interface{}]float64{
		"pizza": -1.0,
		"0.0.1": -1.0,
	}

	for input, _ := range badData {
		_, ok := parameterAsFloat64(input)
		assert.False(t, ok)
	}
}

func TestNormalizedValue(t *testing.T) {
	assert.Equal(t, uint32(73), normalizedValue("123", "gr1"))
	assert.Equal(t, uint32(25), normalizedValue("999", "groupX"))
}

func TestCoalesce(t *testing.T) {
	assert.Equal(t, "", coalesce())
	assert.Equal(t, "foo", coalesce("foo"))
	assert.Equal(t, "bar", coalesce("", "bar"))
}
