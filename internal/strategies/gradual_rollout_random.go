package strategies

import (
	"github.com/Unleash/unleash-client-go/context"
	"github.com/Unleash/unleash-client-go/strategy"
	"math/rand"
	"time"
)

type gradualRolloutRandomStrategy struct {
	random *rand.Rand
}

func NewGradualRolloutRandomStrategy() *gradualRolloutRandomStrategy {
	s := &gradualRolloutRandomStrategy{
		rand.New(rand.NewSource(time.Now().Unix())),
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
