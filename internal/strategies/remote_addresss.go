package strategies

import (
	"net"
	"strings"

	"github.com/Unleash/unleash-go-sdk/v5/context"
	"github.com/Unleash/unleash-go-sdk/v5/strategy"
)

type remoteAddressStrategy struct {
}

func NewRemoteAddressStrategy() *remoteAddressStrategy {
	return &remoteAddressStrategy{}
}

func (s remoteAddressStrategy) Name() string {
	return "remoteAddress"
}

func (s remoteAddressStrategy) IsEnabled(params map[string]any, ctx *context.Context) bool {
	value, found := params[strategy.ParamIps]
	if !found {
		return false
	}

	remoteAddress := net.ParseIP(strings.TrimSpace(ctx.RemoteAddress))
	if remoteAddress == nil {
		return false
	}

	ips, ok := value.(string)
	if !ok {
		return false
	}

	for _, ip := range strings.Split(ips, ",") {
		ip = strings.TrimSpace(ip)
		if remoteAddress.Equal(net.ParseIP(ip)) {
			return true
		} else if _, ipNet, _ := net.ParseCIDR(ip); ipNet != nil && ipNet.Contains(remoteAddress) {
			return true
		}
	}
	return false
}
