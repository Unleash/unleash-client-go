package api

import "time"

type ParameterMap map[string]interface{}

type FeatureResponse struct {
	Response
	Features []Feature `json:"features"`
}

type Feature struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Enabled     bool                `json:"enabled"`
	Strategies  []Strategy          `json:"strategies"`
	CreatedAt   time.Time           `json:"createdAt"`
	Strategy    string              `json:"strategy"`
	Parameters  ParameterMap        `json:"parameters"`
	Variants    []VariantDefinition `json:"variants"`
}

func (fr FeatureResponse) FeatureMap() map[string]interface{} {
	features := map[string]interface{}{}
	for _, f := range fr.Features {
		features[f.Name] = f
	}
	return features
}
