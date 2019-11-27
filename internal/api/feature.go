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
	Name      string         `json:"name"`
	Weight    int32          `json:"weight"`
	Payload   VariantPayload `json:"payload"`
	Overrides []struct {
		ContextName string   `json:"context_name"`
		Values      []string `json:"values"`
	} `json:"overrides"`
}

type VariantPayload struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (fr FeatureResponse) FeatureMap() map[string]interface{} {
	features := map[string]interface{}{}
	for _, f := range fr.Features {
		features[f.Name] = f
	}
	return features
}

func (f Feature) SelectVariant(ctx *context.Context) *VariantPayload {
	// TODO: handle overrides logic
	return &f.Variants[0].Payload
}
