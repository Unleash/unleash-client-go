package strategies

import (
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/Unleash/unleash-client-go/v3/strategy"
)

type gradualRolloutUserId struct {
}

func NewGradualRolloutUserId() *gradualRolloutUserId {
	s := &gradualRolloutUserId{}
	return s
}

func (s gradualRolloutUserId) Name() string {
	return "gradualRolloutUserId"
}

func (s gradualRolloutUserId) IsEnabled(params map[string]interface{}, ctx *context.Context) bool {
	if ctx == nil || ctx.UserID == "" {
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

	normalizedId := normalizedValue(ctx.UserID, groupId)

	return percentage > 0.0 && float64(normalizedId) <= percentage
}
