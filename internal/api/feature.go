package api

import (
	"github.com/konfortes/unleash-client-go/v3/context"
	"time"
)

type ParameterMap map[string]interface{}

type FeatureResponse struct {
	Response
	Features []Feature `json:"features"`
}

type Feature struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Enabled     bool         `json:"enabled"`
	Strategies  []Strategy   `json:"strategies"`
	CreatedAt   time.Time    `json:"createdAt"`
	Strategy    string       `json:"strategy"`
	Parameters  ParameterMap `json:"parameters"`
	Variants    []Variant    `json:"variants"`
}

type Variant struct {
	Name      string            `json:"name"`
	Weight    int32             `json:"weight"`
	Payload   VariantPayload    `json:"payload"`
	Overrides []VariantOverride `json:"overrides"`
}

type VariantPayload struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type VariantOverride struct {
	ContextName string   `json:"context_name"`
	Values      []string `json:"values"`
}

func (fr FeatureResponse) FeatureMap() map[string]interface{} {
	features := map[string]interface{}{}
	for _, f := range fr.Features {
		features[f.Name] = f
	}
	return features
}

// SelectVariant selects the correct variant based on wither context or weights
func (f Feature) SelectVariant(ctx *context.Context) *VariantPayload {

	selectedVariant := f.variantFromOverrides(ctx)
	if selectedVariant == nil {
		selectedVariant = f.variantFromWeights(ctx)
	}

	return &selectedVariant.Payload
}

func (f Feature) variantFromOverrides(ctx *context.Context) *Variant {
	var selectedVariant *Variant
	for _, variant := range f.Variants {
		if variant.matchesContext(ctx) {
			selectedVariant = &variant
		}
	}
	return selectedVariant
}

func (f Feature) variantFromWeights(ctx *context.Context) *Variant {
	// TODO: implement
	return &f.Variants[0]
}

func (v Variant) matchesContext(ctx *context.Context) bool {
	for _, override := range v.Overrides {
		if override.matchesContext(ctx) {
			return true
		}
	}

	return false
}

func (vo VariantOverride) matchesContext(ctx *context.Context) bool {
	var contextValue string
	switch vo.ContextName {
	case "userId":
		contextValue = ctx.UserId
	case "sessionId":
		contextValue = ctx.SessionId
	case "remoteAddress":
		contextValue = ctx.RemoteAddress
	}

	matches := false
	for _, val := range vo.Values {
		if val == contextValue {
			matches = true
		}
	}
	return matches
}
