package strategies

import (
	"github.com/unleash/unleash-client-go/context"
	"github.com/unleash/unleash-client-go/strategy"
)

type gradualRolloutSessionId struct {
}

func NewGradualRolloutSessionId() *gradualRolloutSessionId {
	s := &gradualRolloutSessionId{}
	return s
}

func (s gradualRolloutSessionId) Name() string {
	return "gradualRolloutSessionId"
}

func (s gradualRolloutSessionId) IsEnabled(params map[string]interface{}, ctx *context.Context) bool {
	if ctx == nil || ctx.SessionId == "" {
		return false
	}

	value, found := params[strategy.ParamPercentage]
	if !found {
		return false
	}

	percentage, ok := parameterAsFloat64(value)
	if !ok {
		return false
	}

	groupIdValue := params[strategy.ParamGroupId]
	groupId, ok := groupIdValue.(string)
	if !ok {
		return false
	}

	normalizedId := normalizedValue(ctx.SessionId, groupId)

	return percentage > 0.0 && float64(normalizedId) <= percentage
}
