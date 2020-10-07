package strategy

import "github.com/Unleash/unleash-client-go/context"

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

// EfficientStrategy extends without breaking the Strategy interface.
// To allow initiate more efficient strategy for specific feature.
// For example it can be implemented in user_with_id strategy so CloneEfficient will
// return Strategy that will hold a map of valid user. This way it will not have to
// parse user ids string and run in a loop to search for match on each call.
type EfficientStrategy interface {
	// CloneToEfficient creates new Strategy that uses the parameter in more efficient way.
	CloneEfficient(map[string]interface{}) Strategy
}
