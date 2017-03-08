package strategies

import (
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
		if actual-expected > 0.0000001 {
			t.Errorf("Expected: %v, got %v", expected, actual)
		}
	}

	badData := map[interface{}]float64{
		"pizza": -1.0,
		"0.0.1": -1.0,
	}

	for input, _ := range badData {
		_, ok := parameterAsFloat64(input)
		if ok {
			t.Errorf("Expected: %v to fail", input)
		}

	}
}

func TestNormalizedValue(t *testing.T) {
	norm := normalizedValue("123123", "group1")
	if norm != 2 {
		t.Errorf("Expected: 2, got %v", norm)
	}

}
