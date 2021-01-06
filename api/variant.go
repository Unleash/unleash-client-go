package api

import "github.com/Unleash/unleash-client-go/v3/context"

var DISABLED_VARIANT = &Variant{
	Name:    "disabled",
	Enabled: false,
}

type Payload struct {
	// Type is the type of the payload
	Type string `json:"type"`
	// Value is the value of the payload type
	Value string `json:"value"`
}

type Override struct {
	// ContextName is the value of attribute context name
	ContextName string `json:"contextName"`
	// Values is the value of attribute values
	Values []string `json:"values"`
}

type Variant struct {
	// Name is the value of the variant name.
	Name string `json:"name"`
	// Payload is the value of the variant payload
	Payload Payload `json:"payload"`
	// Enabled indicates whether the feature which is extend by this variant was enabled or not.
	Enabled bool `json:"enabled"`
}

type VariantInternal struct {
	Variant
	// Weight is the traffic ratio for the request
	Weight int `json:"weight"`
	// WeightType can be fixed or variable
	WeightType string `json:"weightType"`
	// Override is used to get a variant accoording to the Unleash context field
	Overrides []Override `json:"overrides"`
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

// Get default variant if no variant is found
func GetDefaultVariant() *Variant {
	return DISABLED_VARIANT
}
