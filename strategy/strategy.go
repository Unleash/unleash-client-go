package strategy

import "github.com/unleash/unleash-client-go/context"

const (
	ParamHostNames  = "hostNames"
	ParamPercentage = "percentage"
	ParamGroupId    = "groupId"
	ParamIps        = "IPs"
	ParamUserIds    = "userIds"
)

type Strategy interface {
	Name() string
	IsEnabled(map[string]interface{}, *context.Context) bool
}
