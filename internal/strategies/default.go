package strategies

import "github.com/Unleash/unleash-go-sdk/v5/context"

type defaultStrategy struct{}

func NewDefaultStrategy() *defaultStrategy {
	return &defaultStrategy{}
}

func (s defaultStrategy) Name() string {
	return "default"
}

func (s defaultStrategy) IsEnabled(_ map[string]any, _ *context.Context) bool {
	return true
}
