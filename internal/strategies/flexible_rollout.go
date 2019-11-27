package strategies

import (
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/Unleash/unleash-client-go/v3/strategy"
)

type stickiness string

const (
	defaultStickiness   stickiness = "default"
	userIDStickiness    stickiness = "userId"
	sessionIDStickiness stickiness = "sessionId"
	randomStickiness    stickiness = "random"
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
	case userIDStickiness:
		return ctx.UserId
	case sessionIDStickiness:
		return ctx.SessionId
	case randomStickiness:
		return s.random.string()
	default:
		return coalesce(ctx.UserId, ctx.SessionId, s.random.string())
	}
}

func (s flexibleRolloutStrategy) IsEnabled(params map[string]interface{}, ctx *context.Context) bool {
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

	sticky := params[strategy.ParamStickiness].(string)
	stickinessID := s.resolveStickiness(stickiness(sticky), *ctx)

	if stickinessID == "" {
		return false
	}

	normalizedID := normalizedValue(stickinessID, groupID)
	return percentage > 0 && float64(normalizedID) <= percentage
}
