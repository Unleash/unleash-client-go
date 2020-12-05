package api

import "github.com/Unleash/unleash-client-go/v3/context"

var DISABLED_VARIANT = &Variant{
	Name: 	 "disabled",
	Enabled: false,
}

type Payload struct {
	Type 	string 	`json:"type"`
	Value 	string	`json:"value"`
}

type Override struct {
	ContextName  string   `json:"contextName"`
	Values       []string `json:"values"`
}

type Variant struct {
	Name 		string		`json:"name"`
	Payload 	Payload 	`json:"payload"`
	Weight 		int			`json:"weight"`
	WeightType  string      `json:"weightType"`
	Overrides   []Override 	`json:"overrides"`
	Enabled     bool		`json:"enabled"`
}

func (f Feature) GetVariant(ctx *context.Context) *Variant {
	if f.Enabled && len(f.Variants) > 0 {
		variant := f.getOverrideVariant(ctx)
		if variant == nil {
			variant = f.getVariantFromWeights(ctx)
		}
		variant.Enabled = true
		return variant
	}
	return DISABLED_VARIANT
}

func (f Feature) getOverrideVariant (ctx *context.Context) *Variant {
	for _, variant := range f.Variants {
		for _, override := range variant.Overrides {
			if override.matchValue(ctx) {
				return &variant
			}
		} 
	}
	return nil
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
	}
	return value
}

func (o Override) matchValue(ctx *context.Context) bool {
	for _, value := range o.Values {
		if value == o.getIdentifier(ctx) {
			return true
		}
	}
	return false
}
