package strategies

import "github.com/Unleash/unleash-client-go/v3/context"

type defaultStrategy struct{}

func NewDefaultStrategy() *defaultStrategy {
	return &defaultStrategy{}
}

func (s defaultStrategy) Name() string {
	return "default"
}

func (s defaultStrategy) IsEnabled(_ map[string]interface{}, _ *context.Context) bool {
	return true
}
