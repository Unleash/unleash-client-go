// +build norace

package strategies

import (
	"github.com/Unleash/unleash-client-go/v4/context"
	"github.com/Unleash/unleash-client-go/v4/strategy"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestGradualRolloutRandomStrategy_Name(t *testing.T) {
	strategy := NewGradualRolloutRandomStrategy()
	assert.Equal(t, "gradualRolloutRandom", strategy.Name(), "strategy should have correct name")
}

func TestGradualRolloutRandomStrategy_IsEnabled(t *testing.T) {
	s := NewGradualRolloutRandomStrategy()

	percentages := []int{0, 10, 33, 50, 87, 100}

	for _, expectedPercentage := range percentages {

		enabledCount := 0
		rounds := 200000

		for i := 0; i < rounds; i++ {
			params := map[string]interface{}{
				strategy.ParamPercentage: expectedPercentage,
				strategy.ParamGroupId:    "groupId",
			}
			enabled := s.IsEnabled(params, &context.Context{
				SessionId: strconv.Itoa(i),
			})
			if enabled {
				enabledCount++
			}
		}

		actualPercentage := round(100.0 * float64(enabledCount) / float64(rounds))

		assert.InDelta(t, expectedPercentage, actualPercentage, 1.0)
	}
}
