package strategy

import "github.com/Unleash/unleash-client-go/v3/context"

const (
	// ParamHostNames is a parameter indicating a comma separated list of hostnames.
	ParamHostNames = "hostNames"

	// ParamPercentage is a property indicating a percentage.
	ParamPercentage = "percentage"

	// ParamGroupId is a property indicating a group ID.
	ParamGroupId = "groupId"

	// ParamIps is a property indicating a comma separated list of ip addresses.
	ParamIps = "IPs"

	// ParamUserIds is a property indicating a comma separated list of user ids.
	ParamUserIds = "userIds"

	// ParamStickiness indicates which context property is sticky.
	ParamStickiness = "stickiness"

	// ParamRollout is a property indicating a rollout percentage.
	ParamRollout = "rollout"
)

// Strategy is an interface for defining new custom strategies.
type Strategy interface {
	// Name should return name of the strategy.
	Name() string

	// IsEnabled should look at the map of parameters and optionally
	// the supplied context and return true if the feature should be
	// enabled.
	IsEnabled(map[string]interface{}, *context.Context) bool
}
