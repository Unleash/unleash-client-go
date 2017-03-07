package strategies

import (
	"github.com/unleash/unleash-client-go/context"
	"github.com/unleash/unleash-client-go/strategy"
	"strings"
)

type remoteAddressStrategy struct {
}

func NewRemoteAddressStrategy() *remoteAddressStrategy {
	return &remoteAddressStrategy{}
}

func (s remoteAddressStrategy) Name() string {
	return "remoteAddress"
}

func (s remoteAddressStrategy) IsEnabled(params map[string]interface{}, ctx *context.Context) bool {
	value, found := params[strategy.ParamIps]
	if !found {
		return false
	}

	ips, ok := value.(string)
	if !ok {
		return false
	}

	for _, ip := range strings.Split(ips, ",") {
		if strings.TrimSpace(ip) == ctx.RemoteAddress {
			return true
		}
	}
	return false
}
