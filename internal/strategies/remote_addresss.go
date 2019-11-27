package strategies

import (
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/Unleash/unleash-client-go/v3/strategy"
	"net"
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
