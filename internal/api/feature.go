package api

import (
	"github.com/Unleash/unleash-client-go/strategy"
	"time"
)

type ParameterMap map[string]interface{}

type FeatureResponse struct {
	Response
	Features []Feature `json:"features"`
}

type SupportedStrategies struct {
	FeatureStrategy Strategy
	ClientStrategy  strategy.Strategy
}

type Feature struct {
	Name                string       `json:"name"`
	Description         string       `json:"description"`
	Enabled             bool         `json:"enabled"`
	Strategies          []Strategy   `json:"strategies"`
	CreatedAt           time.Time    `json:"createdAt"`
	Strategy            string       `json:"strategy"`
	Parameters          ParameterMap `json:"parameters"`
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
