package unleash

import (
	"fmt"
	"slices"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/Unleash/unleash-go-sdk/v6/context"
	"github.com/Unleash/unleash-go-sdk/v6/internal/constraints"
	"github.com/Unleash/unleash-go-sdk/v6/strategy"
)

type FeatureMemoryState struct {
	Features map[string]*api.Feature
	Segments map[int][]api.Constraint
}

func (s *FeatureMemoryState) list() []api.Feature {
	features := make([]api.Feature, 0, len(s.Features))

	// we're doing an explicit copy here, this function should not be on a hot path
	for _, feature := range s.Features {
		if feature == nil {
			continue
		}
		features = append(features, *feature)
	}
	return features
}

func (s *FeatureMemoryState) asApiResponse() *api.FeatureResponse {
	features := make([]api.Feature, 0, len(s.Features))
	for _, f := range s.Features {
		features = append(features, *f)
	}

	segments := make([]api.Segment, 0, len(s.Segments))
	for id, constraints := range s.Segments {
		segments = append(segments, api.Segment{
			Id:          id,
			Constraints: constraints,
		})
	}

	return &api.FeatureResponse{
		Features: features,
		Segments: segments,
	}
}

// evaluateFeature applies strategies + constraints for a single feature.
// It does NOT handle fallbacks or missing features; that's the caller's job.
func (s *FeatureMemoryState) evaluateFeature(
	f *api.Feature,
	ctx *context.Context,
	strategies []strategy.Strategy,
) (api.StrategyResult, error) {

	if f.Dependencies != nil && len(*f.Dependencies) > 0 {
		dependenciesSatisfied := s.isParentDependencySatisfied(f, ctx, strategies)

		if !dependenciesSatisfied {
			return api.StrategyResult{
				Enabled: false,
			}, nil
		}
	}

	if !f.Enabled {
		return api.StrategyResult{
			Enabled: false,
		}, nil
	}

	if len(f.Strategies) == 0 {
		return api.StrategyResult{
			Enabled: f.Enabled,
		}, nil
	}

	for _, stCfg := range f.Strategies {
		foundStrategy := findStrategy(strategies, stCfg.Name)
		if foundStrategy == nil {
			// TODO: warnOnce missing strategy
			continue
		}

		segmentConstraints, err := s.resolveSegmentConstraints(stCfg)
		if err != nil {
			return api.StrategyResult{
				Enabled: false,
			}, err
		}

		allConstraints := make([]api.Constraint, 0, len(segmentConstraints)+len(stCfg.Constraints))
		allConstraints = append(allConstraints, segmentConstraints...)
		allConstraints = append(allConstraints, stCfg.Constraints...)

		ok, _ := constraints.Check(ctx, allConstraints)
		if !ok {
			continue
		}

		if !foundStrategy.IsEnabled(stCfg.Parameters, ctx) {
			continue
		}

		if stCfg.Variants != nil && len(stCfg.Variants) > 0 {
			groupIdValue := stCfg.Parameters[strategy.ParamGroupId]
			groupId, ok := groupIdValue.(string)
			if !ok {
				return api.StrategyResult{
					Enabled: false,
				}, fmt.Errorf("invalid groupId type for feature %s", f.Name)
			}

			var stickiness *string

			if v, ok := stCfg.Parameters[strategy.ParamStickiness]; ok {
				if s, ok := v.(string); ok {
					stickiness = &s
				}
			}

			return api.StrategyResult{
				Enabled: true,
				Variant: api.VariantCollection{
					GroupId:  groupId,
					Variants: stCfg.Variants,
				}.GetVariant(ctx, stickiness),
			}, nil
		}

		return api.StrategyResult{
			Enabled: true,
		}, nil
	}

	return api.StrategyResult{
		Enabled: false,
	}, nil
}

func (s *FeatureMemoryState) isParentDependencySatisfied(feature *api.Feature, ctx *context.Context, strategies []strategy.Strategy) bool {
	warnOnce := &WarnOnce{}

	dependenciesSatisfied := func(parent api.Dependency) bool {
		parentToggle := s.Features[parent.Feature]

		if parentToggle == nil {
			warnOnce.Warn("the parent toggle was not found in the cache, the evaluation of this dependency will always be false")
			return false
		}

		if parentToggle.Dependencies != nil && len(*parentToggle.Dependencies) > 0 {
			return false
		}

		enabledResult, err := s.evaluateFeature(parentToggle, ctx, strategies)
		if err != nil {
			warnOnce.Warn("error evaluating parent toggle, the evaluation of this dependency will always be false")
			return false
		}
		// According to the schema, if the enabled property is absent we assume it's true.
		if parent.Enabled == nil || *parent.Enabled {
			if parent.Variants != nil && len(*parent.Variants) > 0 && enabledResult.Variant != nil {
				return enabledResult.Enabled && slices.Contains(*parent.Variants, enabledResult.Variant.Name)
			}
			return enabledResult.Enabled
		}

		return !enabledResult.Enabled
	}

	allDependenciesSatisfied := every(*feature.Dependencies, func(parent api.Dependency) bool {
		return dependenciesSatisfied(parent)
	})

	return allDependenciesSatisfied
}

func findStrategy(strats []strategy.Strategy, name string) strategy.Strategy {
	for _, s := range strats {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

func (featureState *FeatureMemoryState) resolveSegmentConstraints(strategy api.Strategy) ([]api.Constraint, error) {
	segmentConstraints := []api.Constraint{}

	segments := featureState.Segments

	for _, segmentId := range strategy.Segments {
		if resolvedConstraints, ok := segments[segmentId]; ok {
			segmentConstraints = append(segmentConstraints, resolvedConstraints...)
		} else {
			return segmentConstraints, fmt.Errorf("segment does not exist")
		}
	}

	return segmentConstraints, nil
}
