package strategies

import (
	"testing"

	"github.com/Unleash/unleash-go-sdk/v6/context"
	"github.com/stretchr/testify/assert"
)

func TestDefaultStrategy_Name(t *testing.T) {
	strategy := NewDefaultStrategy()
	assert.Equal(t, "default", strategy.Name(), "strategy should have correct name")
}

func TestDefaultStrategy_IsEnabled(t *testing.T) {
	s := NewDefaultStrategy()
	assert.True(t, s.IsEnabled(nil, context.Context{}), "default strategy should be enabled")
}
