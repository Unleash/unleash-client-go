package strategies

import (
	"math/rand"
	"strconv"
	"time"

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
	random *rand.Rand
}

// NewFlexibleRolloutStrategy creates a new instance of the flexible rollout strategy.
func NewFlexibleRolloutStrategy() *flexibleRolloutStrategy {
	s := &flexibleRolloutStrategy{
		random: rand.New(rand.NewSource(time.Now().Unix())),
	}
	return s
}

func (s flexibleRolloutStrategy) Name() string {
	return "flexibleRollout"
}

func (s flexibleRolloutStrategy) randomNumber() string {
	return strconv.Itoa(s.random.Intn(100) + 1)
}

func (s flexibleRolloutStrategy) resolveStickiness(st stickiness, ctx context.Context) string {
	switch st {
	case userIDStickiness:
		return ctx.UserId
	case sessionIDStickiness:
		return ctx.SessionId
	case randomStickiness:
		return s.randomNumber()
	default:
		return coalesce(ctx.UserId, ctx.SessionId, s.randomNumber())
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
