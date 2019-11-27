package strategies

import (
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/Unleash/unleash-client-go/v3/strategy"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestGradualRolloutSessionId_Name(t *testing.T) {
	strategy := NewGradualRolloutSessionId()
	assert.Equal(t, "gradualRolloutSessionId", strategy.Name(), "strategy should have correct name")
}

func TestGradualRolloutSessionId_IsEnabled(t *testing.T) {
	s := NewGradualRolloutSessionId()
	assert := assert.New(t)

	t.Run("p=100", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamPercentage: 100,
			strategy.ParamGroupId:    "gr1",
		}
		isEnabled := s.IsEnabled(params, &context.Context{
			SessionId: "123",
		})
		assert.True(isEnabled, "should be enabled when percentage is 100")
	})

	t.Run("p=0", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamPercentage: 0,
			strategy.ParamGroupId:    "gr1",
		}
		isEnabled := s.IsEnabled(params, &context.Context{
			SessionId: "123",
		})
		assert.False(isEnabled, "should be disabled when percentage is 0")
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
		assert.True(isEnabled, "should be enabled when percentage is exactly same")
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

		assert.False(isEnabled, "should be disabled when percentage is just below required value")
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

			assert.InDelta(expectedPercentage, actualPercentage, 1.0)
		}
	})
}
