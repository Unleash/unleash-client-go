package api

import (
	"time"
	"math/rand"
	"strconv"
	"fmt"

	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/spaolacci/murmur3"
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
	Variants	[]Variant    `json:"variants"`
}

func (fr FeatureResponse) FeatureMap() map[string]interface{} {
	features := map[string]interface{}{}
	for _, f := range fr.Features {
		features[f.Name] = f
	}
	return features
}

func (f Feature) getVariantFromWeights(ctx *context.Context) *Variant {
	if len(f.Variants) > 0 {
		totalWeight := 0
		for _, variant := range f.Variants {
			totalWeight += variant.Weight
		}
		if totalWeight == 0 {
			return DISABLED_VARIANT
		}

		target := getNormalizedNumber(getSeed(ctx), f.Name, totalWeight)

		counter := uint32(0)
		for _, variant := range f.Variants {
			counter += uint32(variant.Weight)
			
			if counter >= target {
				return &variant
			}
		}
	}
	return DISABLED_VARIANT
}

func getSeed(ctx *context.Context) string {
	if ctx.UserId != "" {
		return ctx.UserId
	} else if ctx.SessionId != "" {
		return ctx.SessionId
	} else if ctx.RemoteAddress != "" {
		return ctx.RemoteAddress
	} else {
		return strconv.Itoa(rand.Intn(10000))
	}
}

func getNormalizedNumber(identifier string, groupId string, normalizer int) uint32 {
	return (murmur3.Sum32([]byte(fmt.Sprintf("%s:%s", identifier, groupId))) % uint32(normalizer)) + 1
}
