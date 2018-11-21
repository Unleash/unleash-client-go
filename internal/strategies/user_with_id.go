package strategies

import (
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/Unleash/unleash-client-go/v3/strategy"
	"strings"
)

type userWithIdStrategy struct{}

func NewUserWithIdStrategy() *userWithIdStrategy {
	return &userWithIdStrategy{}
}

func (s userWithIdStrategy) Name() string {
	return "userWithId"
}

func (s userWithIdStrategy) IsEnabled(params map[string]interface{}, ctx *context.Context) bool {
	value, found := params[strategy.ParamUserIds]
	if !found {
		return false
	}

	userIds, ok := value.(string)
	if !ok {
		return false
	}

	for _, u := range strings.Split(userIds, ",") {
		if strings.TrimSpace(u) == ctx.UserId {
			return true
		}
	}
	return false
}
