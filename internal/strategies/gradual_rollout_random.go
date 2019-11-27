package strategies

import (
	"github.com/konfortes/unleash-client-go/v3/context"
	"github.com/konfortes/unleash-client-go/v3/strategy"
)

type gradualRolloutRandomStrategy struct {
	random *rng
}

func NewGradualRolloutRandomStrategy() *gradualRolloutRandomStrategy {
	s := &gradualRolloutRandomStrategy{
		newRng(),
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

	return percentage >= s.random.float()
}
