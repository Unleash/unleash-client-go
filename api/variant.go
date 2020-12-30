package api

import "github.com/Unleash/unleash-client-go/v3/context"

var DISABLED_VARIANT = &Variant{
	Name:    "disabled",
	Enabled: false,
}

type Payload struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Override struct {
	ContextName string   `json:"contextName"`
	Values      []string `json:"values"`
}

type Variant struct {
	Name       string     `json:"name"`
	Payload    Payload    `json:"payload"`
	Weight     int        `json:"weight"`
	WeightType string     `json:"weightType"`
	Overrides  []Override `json:"overrides"`
	Enabled    bool       `json:"enabled"`
}

func (o Override) getIdentifier(ctx *context.Context) string {
	var value string
	switch o.ContextName {
	case "userId":
		value = ctx.UserId
	case "sessionId":
		value = ctx.SessionId
	case "remoteAddress":
		value = ctx.RemoteAddress
	default:
		if len(ctx.Properties) > 0 {
			for k, v := range ctx.Properties {
				if k == o.ContextName {
					value = v
				}
			}
		}
	}
	return value
}

func (o Override) matchValue(ctx *context.Context) bool {
	if len(o.Values) == 0 {
		return false
	}
	for _, value := range o.Values {
		if value == o.getIdentifier(ctx) {
			return true
		}
	}
	return false
}

func GetDefaultVariant() *Variant {
	return DISABLED_VARIANT
}