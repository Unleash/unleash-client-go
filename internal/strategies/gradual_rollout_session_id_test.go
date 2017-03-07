package strategies

import (
	"github.com/unleash/unleash-client-go/context"
	"github.com/unleash/unleash-client-go/strategy"
	"strconv"
	"testing"
)

func TestGradualRolloutSessionId_Name(t *testing.T) {
	strategy := NewGradualRolloutSessionId()

	if strategy.Name() != "gradualRolloutSessionId" {
		t.Errorf("gradual-rollout-user-id strategy should have correct name: %s", strategy.Name())
	}
}

func TestGradualRolloutSessionId_IsEnabled(t *testing.T) {

	s := NewGradualRolloutSessionId()

	t.Run("p=100", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamPercentage: 100,
			strategy.ParamGroupId:    "gr1",
		}
		isEnabled := s.IsEnabled(params, &context.Context{
			SessionId: "123",
		})
		if !isEnabled {
			t.Error("should be enabled when percentage is 100")
		}
	})

	t.Run("p=0", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamPercentage: 0,
			strategy.ParamGroupId:    "gr1",
		}
		isEnabled := s.IsEnabled(params, &context.Context{
			SessionId: "123",
		})
		if isEnabled {
			t.Error("should be disabled when percentage is 0")
		}
	})

	t.Run("p1=p2", func(t *testing.T) {
		sessionId := "123123"
		groupId := "group1"
		percentage := normalizedValue(sessionId, groupId)

		params := map[string]interface{}{
			strategy.ParamPercentage: percentage,
			strategy.ParamGroupId:    groupId,
		}
		isEnabled := s.IsEnabled(params, &context.Context{
			SessionId: sessionId,
		})
		if !isEnabled {
			t.Error("should be enabled when percentage is exactly same")
		}
	})

	t.Run("p1<p2", func(t *testing.T) {
		sessionId := "123123"
		groupId := "group1"
		percentage := normalizedValue(sessionId, groupId) - 1

		params := map[string]interface{}{
			strategy.ParamPercentage: percentage,
			strategy.ParamGroupId:    groupId,
		}
		isEnabled := s.IsEnabled(params, &context.Context{
			SessionId: sessionId,
		})
		if isEnabled {
			t.Error("should be disabled when percentage is just below required value")
		}
	})

	t.Run("p1-p1<1", func(t *testing.T) {
		percentages := []int{0, 10, 33, 50, 87, 100}

		for _, expectedPercentage := range percentages {

			enabledCount := 0
			rounds := 200000

			for i := 0; i < rounds; i++ {
				params := map[string]interface{}{
					strategy.ParamPercentage: expectedPercentage,
					strategy.ParamGroupId:    "gr1",
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
	})
}
