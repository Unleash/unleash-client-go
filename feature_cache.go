package unleash

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Unleash/unleash-go-sdk/v6/api"
)

type featureCache struct {
	featureState atomic.Value
	mu           sync.Mutex
}

func newFeatureCache(baseState *api.FeatureResponse) *featureCache {

	if baseState == nil {
		baseState = &api.FeatureResponse{
			Features: make([]api.Feature, 0),
			Segments: make([]api.Segment, 0),
		}
	}

	featureState := &FeatureMemoryState{
		Features: baseState.FeatureMap(),
		Segments: baseState.SegmentsMap(),
	}

	featureCache := &featureCache{
		featureState: atomic.Value{},
	}

	featureCache.featureState.Store(featureState)
	return featureCache
}

func (dp *featureCache) updateFromApiResponse(api *api.ApiResponse) (*api.FeatureResponse, error) {
	if api.IsFullResponse() {
		return dp.updateFromFullResponse(api.Full)
	}

	if api.IsDeltaResponse() {
		return dp.updateFromDelta(api.Delta)
	}

	return nil, fmt.Errorf("unknown API response type")
}

func (dp *featureCache) updateFromFullResponse(full *api.FeatureResponse) (*api.FeatureResponse, error) {
	newState := &FeatureMemoryState{
		Features: full.FeatureMap(),
		Segments: full.SegmentsMap(),
	}

	dp.featureState.Store(newState)
	return full, nil
}

// It's hard to make this both concurrency safe and atomic for both readers and writes. To build our new state we need
// to hold a lock over the entire process to prevent internal races setting the final state out of order.
// However, once we have built the new state we can atomically swap it. This gives us mutex locked writes but
// lock free reads via snapshot()
func (dp *featureCache) updateFromDelta(delta *api.ClientFeaturesDelta) (*api.FeatureResponse, error) {
	if delta == nil {
		return nil, fmt.Errorf("delta is nil")
	}

	dp.mu.Lock()
	defer dp.mu.Unlock()

	snap := dp.snapshot()

	featMap := make(map[string]*api.Feature, len(snap.Features))
	for name, f := range snap.Features {
		featMap[name] = f
	}

	segMap := make(map[int][]api.Constraint, len(snap.Segments))
	for id, constraints := range snap.Segments {
		segMap[id] = constraints
	}

	deltaState := &api.DeltaState{
		Features: featMap,
		Segments: segMap,
	}

	for _, event := range delta.Events {
		event.Apply(deltaState)
	}

	newState := &FeatureMemoryState{
		Features: deltaState.Features,
		Segments: deltaState.Segments,
	}

	dp.featureState.Store(newState)

	return newState.asApiResponse(), nil
}

func (dp *featureCache) snapshot() *FeatureMemoryState {
	v := dp.featureState.Load()
	return v.(*FeatureMemoryState)
}
