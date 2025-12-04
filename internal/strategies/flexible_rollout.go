package strategies

import (
	"github.com/Unleash/unleash-go-sdk/v6/context"
	"github.com/Unleash/unleash-go-sdk/v6/strategy"
)

type stickiness string

const (
	defaultStickiness stickiness = "default"
	randomStickiness  stickiness = "random"
)

type flexibleRolloutStrategy struct {
	random *rng
}

// NewFlexibleRolloutStrategy creates a new instance of the flexible rollout strategy.
func NewFlexibleRolloutStrategy() *flexibleRolloutStrategy {
	s := &flexibleRolloutStrategy{
		random: newRng(),
	}
	return s
}

func (s flexibleRolloutStrategy) Name() string {
	return "flexibleRollout"
}

func (s flexibleRolloutStrategy) resolveStickiness(st stickiness, ctx context.Context) string {
	switch st {
	case defaultStickiness:
		return coalesce(ctx.UserId, ctx.SessionId, randomString())
	case randomStickiness:
		return randomString()
	default:
		return ctx.Field(string(st))
	}
}

func (s flexibleRolloutStrategy) IsEnabled(params map[string]any, ctx *context.Context) bool {
	groupID := ""
	if gID, found := params[strategy.ParamGroupId]; found {
		groupID = gID.(string)
	}

	rollout, found := params[strategy.ParamRollout]
	if !found {
		return false
	}

	percentage, ok := parameterAsFloat64(rollout)
	if !ok {
		return false
	}

	sticky := coalesce(params[strategy.ParamStickiness].(string), string(defaultStickiness))
	stickinessID := s.resolveStickiness(stickiness(sticky), *ctx)

	if stickinessID == "" {
		return false
	}

	normalizedID := normalizedRolloutValue(stickinessID, groupID)
	return percentage > 0 && float64(normalizedID) <= percentage
}
