package strategies

import (
	"testing"

	"github.com/Unleash/unleash-client-go/v4/context"
	"github.com/Unleash/unleash-client-go/v4/strategy"
	"github.com/stretchr/testify/assert"
)

func TestFlexibleRolloutStrategy_IsWellDistributed(t *testing.T) {
	s := NewFlexibleRolloutStrategy()

	enabledCount := 0
	rounds := 200000

	for i := 0; i < rounds; i++ {
		params := map[string]interface{}{
			strategy.ParamStickiness: "random",
			strategy.ParamRollout:    50,
			strategy.ParamGroupId:    "test51",
		}
		enabled := s.IsEnabled(params, &context.Context{})
		if enabled {
			enabledCount++
		}
	}

	actualPercentage := round(100.0 * float64(enabledCount) / float64(rounds))

	assert.InDelta(t, 50, actualPercentage, 1.0)
}
