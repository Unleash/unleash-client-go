package strategies

import (
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/Unleash/unleash-client-go/v3/strategy"
)

type gradualRolloutSessionID struct {
}

func NewGradualRolloutSessionId() *gradualRolloutSessionID {
	s := &gradualRolloutSessionID{}
	return s
}

func (s gradualRolloutSessionID) Name() string {
	return "gradualRolloutSessionID"
}

func (s gradualRolloutSessionID) IsEnabled(params map[string]interface{}, ctx *context.Context) bool {
	if ctx == nil || ctx.SessionID == "" {
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

	groupIdValue := params[strategy.ParamGroupID]
	groupId, ok := groupIdValue.(string)
	if !ok {
		return false
	}

	normalizedId := normalizedValue(ctx.SessionID, groupId)

	return percentage > 0.0 && float64(normalizedId) <= percentage
}
