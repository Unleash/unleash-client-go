package api

import "github.com/Unleash/unleash-client-go/v4/context"

var DISABLED_VARIANT = &Variant{
	Name:           "disabled",
	Enabled:        false,
	FeatureEnabled: false,
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
	// Enabled indicates whether the variant is enabled. This is only false when
	// it's a default variant.
	Enabled bool `json:"enabled"`
	// FeatureEnabled indicates whether the Feature for this variant is enabled.
	FeatureEnabled bool `json:"featureEnabled"`
}

type VariantInternal struct {
	Variant
	// Weight is the traffic ratio for the request
	Weight int `json:"weight"`
	// WeightType can be fixed or variable
	WeightType string `json:"weightType"`
	Stickiness string `json:"stickiness"`
	// Override is used to get a variant accoording to the Unleash context field
	Overrides []Override `json:"overrides"`
}

type VariantCollection struct {
	// groupId to evaluate the variant
	GroupId string
	// variants for a feature toggle or feature strategy
	Variants []VariantInternal
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

// Get default variant if feature is not found or if the feature is disabled.
//
// Rather than checking against this particular variant you should be checking
// the returned variant's Enabled and FeatureEnabled properties.
func GetDefaultVariant() *Variant {
	return DISABLED_VARIANT
}
