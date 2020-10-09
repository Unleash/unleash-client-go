package api

import (
	"time"

	"github.com/Unleash/unleash-client-go/v3/strategy"
)

type ParameterMap map[string]interface{}

type FeatureResponse struct {
	Response
	Features []Feature `json:"features"`
}

type SupportedStrategies struct {
	FeatureStrategy     Strategy
	ClientStrategy      strategy.Strategy
	SupportedStrategies []SupportedStrategies
}

type Feature struct {
	// Name is the name of the feature toggle.
	Name string `json:"name"`

	// Description is a description of the feature toggle.
	Description string `json:"description"`

	// Enabled indicates whether the feature was enabled or not.
	Enabled bool `json:"enabled"`

	// Strategies is a list of names of the strategies supported by the client.
	Strategies []Strategy `json:"strategies"`

	// CreatedAt is the creation time of the feature toggle.
	CreatedAt time.Time `json:"createdAt"`

	// Strategy is the strategy of the feature toggle.
	Strategy string `json:"strategy"`

	// Parameters is the parameters of the feature toggle.
	Parameters ParameterMap `json:"parameters"`

	// SupportedStrategies are the client strategies that supports this feature
	SupportedStrategies []SupportedStrategies
}

func (fr FeatureResponse) FeatureMap(clientStrategies []strategy.Strategy) map[string]interface{} {
	features := map[string]interface{}{}
	for _, f := range fr.Features {
		f.initFeature(clientStrategies)
		features[f.Name] = f
	}
	return features
}

func (f *Feature) initFeature(clientStrategies []strategy.Strategy) {
	f.SupportedStrategies = make([]SupportedStrategies, 0, len(f.Strategies))

	for _, s := range f.Strategies {
		supportedStrategy := getStrategy(s, clientStrategies)
		if supportedStrategy == nil {
			// TODO: warnOnce missingStrategy
			continue
		}

		f.SupportedStrategies = append(f.SupportedStrategies, *supportedStrategy)
	}
}

func getStrategy(featureStrategy Strategy, clientStrategies []strategy.Strategy) *SupportedStrategies {
	for _, clientStrategy := range clientStrategies {
		if clientStrategy.Name() == featureStrategy.Name {
			if adoptableStrategy, ok := clientStrategy.(strategy.EfficientStrategy); ok {
				clientStrategy = adoptableStrategy.CloneEfficient(featureStrategy.Parameters)
			}

			if clientStrategy == nil {
				return nil
			}

			return &SupportedStrategies{
				FeatureStrategy: featureStrategy,
				ClientStrategy:  clientStrategy,
			}
		}
	}
	return nil
}
