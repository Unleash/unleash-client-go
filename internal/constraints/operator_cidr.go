package constraints

import (
	"net"
	"strings"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/Unleash/unleash-go-sdk/v6/context"
)

func operatorInCidr(ctx *context.Context, constraint api.Constraint) bool {
	contextValue := strings.TrimSpace(ctx.Field(constraint.ContextName))
	if contextValue == "" {
		return false
	}

	remoteAddress := net.ParseIP(contextValue)
	if remoteAddress == nil {
		return false
	}

	for _, value := range constraint.Values {
		ipOrCidr := strings.TrimSpace(value)
		if ipOrCidr == "" {
			continue
		}

		if remoteAddress.Equal(net.ParseIP(ipOrCidr)) {
			return true
		}

		if _, ipNet, err := net.ParseCIDR(ipOrCidr); err == nil && ipNet != nil && ipNet.Contains(remoteAddress) {
			return true
		}
	}

	return false
}
