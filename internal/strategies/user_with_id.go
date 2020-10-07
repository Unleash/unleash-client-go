package strategies

import (
	"github.com/Unleash/unleash-client-go/context"
	"github.com/Unleash/unleash-client-go/strategy"
	"strings"
)

type userWithIdStrategy struct {
	userIds map[string]bool
}

func NewUserWithIdStrategy() *userWithIdStrategy {
	return &userWithIdStrategy{}
}

func (s userWithIdStrategy) Name() string {
	return "userWithId"
}

func (s userWithIdStrategy) IsEnabled(_ map[string]interface{}, ctx *context.Context) bool {
	_, ok := s.userIds[ctx.UserId]
	return ok
}

func (s userWithIdStrategy) CloneEfficient(params map[string]interface{}) strategy.Strategy {
	value, found := params[strategy.ParamUserIds]
	if !found {
		return nil
	}

	userIds, ok := value.(string)
	if !ok {
		return nil
	}

	usersMap := make(map[string]bool, len(s.userIds))
	for _, u := range strings.Split(userIds, ",") {
		usersMap[strings.TrimSpace(u)] = true
	}

	return &userWithIdStrategy{userIds: usersMap}
}
