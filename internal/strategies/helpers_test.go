package strategies

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
	norm := normalizedValue("123123", "group1")
	assert.Equal(t, int64(2), norm)
}
