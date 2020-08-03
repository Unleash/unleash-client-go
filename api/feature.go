package api

import (
	"time"
)

type ParameterMap map[string]interface{}

type FeatureResponse struct {
	Response
	Features []Feature `json:"features"`
}

type Feature struct {
	// Name is the name of the feature toggle.
	Name        string       `json:"name"`

	// Description is a description of the feature toggle.
	Description string       `json:"description"`

	// Enabled indicates whether the feature was enabled or not.
	Enabled     bool         `json:"enabled"`

	// Strategies is a list of names of the strategies supported by the client.
	Strategies  []Strategy   `json:"strategies"`

	// CreatedAt is the creation time of the feature toggle.
	CreatedAt   time.Time    `json:"createdAt"`

	// Strategy is the strategy of the feature toggle.
	Strategy    string       `json:"strategy"`

	// Parameters is the parameters of the feature toggle.
	Parameters  ParameterMap `json:"parameters"`
}

func (fr FeatureResponse) FeatureMap() map[string]interface{} {
	features := map[string]interface{}{}
	for _, f := range fr.Features {
		features[f.Name] = f
	}
	return features
}
