package api

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/twmb/murmur3"
)

type ParameterMap map[string]interface{}

type FeatureResponse struct {
	Response
	Features []Feature `json:"features"`
	Segments []Segment `json:"segments"`
}

type Segment struct {
	Id          int          `json:"id"`
	Constraints []Constraint `json:"constraints"`
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

	// Variants is a list of variants of the feature toggle.
	Variants []VariantInternal `json:"variants"`

	// Dependencies is a list of feature toggle dependency objects
	Dependencies *[]FeatureDependencies `json:"dependencies"`
}

type FeatureDependencies struct {
	// Feature is the name of the feature toggle we depend upon
	Feature string `json:"feature"`
	// Variants contains a string of variants that the dependency should resolve to
	Variants *[]string `json:"variants"`
	// Enabled is the property that determines whether the dependency should be on or off
	// If the property is absent from the payload it's assumed to be default on
	Enabled *bool `json:"enabled"`
}

func (fr FeatureResponse) FeatureMap() map[string]interface{} {
	features := map[string]interface{}{}
	for _, f := range fr.Features {
		features[f.Name] = f
	}
	return features
}

func (fr FeatureResponse) SegmentsMap() map[int][]Constraint {
	segments := map[int][]Constraint{}
	for _, segment := range fr.Segments {
		segments[segment.Id] = segment.Constraints
	}

	return segments
}

// Get variant for a given feature which is considered as enabled
func (vc VariantCollection) GetVariant(ctx *context.Context) *Variant {
	if len(vc.Variants) > 0 {
		v := vc.getOverrideVariant(ctx)
		var variant *Variant
		if v == nil {
			variant = vc.getVariantFromWeights(ctx)
		} else {
			variant = &v.Variant
		}
		variant.Enabled = true
		return variant
	}
	return DISABLED_VARIANT
}

func (vc VariantCollection) getVariantFromWeights(ctx *context.Context) *Variant {
	totalWeight := 0
	for _, variant := range vc.Variants {
		totalWeight += variant.Weight
	}
	if totalWeight == 0 {
		return DISABLED_VARIANT
	}
	stickiness := vc.Variants[0].Stickiness

	target := getNormalizedNumber(getSeed(ctx, stickiness), vc.GroupId, totalWeight)
	counter := uint32(0)
	for _, variant := range vc.Variants {
		counter += uint32(variant.Weight)

		if counter >= target {
			return &variant.Variant
		}
	}
	return DISABLED_VARIANT
}

func (vc VariantCollection) getOverrideVariant(ctx *context.Context) *VariantInternal {
	for _, variant := range vc.Variants {
		for _, override := range variant.Overrides {
			if override.matchValue(ctx) {
				variant.Overrides = nil
				return &variant
			}
		}
	}
	return nil
}

func getSeed(ctx *context.Context, stickiness string) string {
	if stickiness != "default" && stickiness != "" {
		value := ctx.Field(stickiness)
		if value == "" {
			return strconv.Itoa(rand.Intn(10000))
		}

		return value
	}

	if ctx.UserId != "" {
		return ctx.UserId
	} else if ctx.SessionId != "" {
		return ctx.SessionId
	} else if ctx.RemoteAddress != "" {
		return ctx.RemoteAddress
	}
	return strconv.Itoa(rand.Intn(10000))
}

func getNormalizedNumber(identifier, groupId string, normalizer int) uint32 {
	return (murmur3.Sum32([]byte(fmt.Sprintf("%s:%s", groupId, identifier))) % uint32(normalizer)) + 1
}
