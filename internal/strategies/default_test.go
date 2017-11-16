package strategies

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultStrategy_Name(t *testing.T) {
	strategy := NewDefaultStrategy()
	assert.Equal(t, "default", strategy.Name(), "strategy should have correct name")
}

func TestDefaultStrategy_IsEnabled(t *testing.T) {
	s := NewDefaultStrategy()
	assert.True(t, s.IsEnabled(nil, nil), "default strategy should be enabled")
}
