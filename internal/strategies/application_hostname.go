package strategies

import (
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/Unleash/unleash-client-go/v3/strategy"
	"strings"
)

type applicationHostnameStrategy struct {
	hostname string
}

func NewApplicationHostnameStrategy() *applicationHostnameStrategy {
	hostname, _ := resolveHostname()
	return &applicationHostnameStrategy{
		strings.ToLower(hostname),
	}
}

func (s applicationHostnameStrategy) Name() string {
	return "applicationHostname"
}

func (s applicationHostnameStrategy) IsEnabled(params map[string]interface{}, _ *context.Context) bool {
	value, found := params[strategy.ParamHostNames]
	if !found {
		return false
	}

	hostnames, ok := value.(string)
	if !ok {
		return false
	}

	for _, h := range strings.Split(hostnames, ",") {
		if strings.ToLower(h) == s.hostname {
			return true
		}
	}
	return false
}
