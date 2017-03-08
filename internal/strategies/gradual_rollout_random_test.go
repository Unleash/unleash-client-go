package strategies

import (
	"github.com/Unleash/unleash-client-go/context"
	"github.com/Unleash/unleash-client-go/strategy"
	"strconv"
	"testing"
)

func TestGradualRolloutRandomStrategy_Name(t *testing.T) {
	strategy := NewGradualRolloutRandomStrategy()

	if strategy.Name() != "gradualRolloutRandom" {
		t.Errorf("strategy should have correct name: %s", strategy.Name())
	}
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
		highMark := expectedPercentage + 1
		lowMark := expectedPercentage - 1

		if actualPercentage < lowMark || actualPercentage > highMark {
			t.Errorf("Expected percentage of %d, got %d", expectedPercentage, actualPercentage)
		}
	}
}
