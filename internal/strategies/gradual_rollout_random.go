package strategies

import (
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/Unleash/unleash-client-go/v3/strategy"
	"math/rand"
)

type gradualRolloutRandomStrategy struct {
	random *rand.Rand
}

func NewGradualRolloutRandomStrategy() *gradualRolloutRandomStrategy {
	s := &gradualRolloutRandomStrategy{
		newRand(),
	}
	return s
}

func (s gradualRolloutRandomStrategy) Name() string {
	return "gradualRolloutRandom"
}

func (s gradualRolloutRandomStrategy) IsEnabled(params map[string]interface{}, _ *context.Context) bool {
	value, found := params[strategy.ParamPercentage]
	if !found {
		return false
	}

	percentage, ok := parameterAsFloat64(value)
	if !ok {
		return false
	}

	return percentage >= float64(s.random.Intn(100)+1)
}
