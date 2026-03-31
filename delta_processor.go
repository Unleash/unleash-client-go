package unleash

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Unleash/unleash-go-sdk/v6/api"
)

type deltaProcessor struct {
	featureState atomic.Value
	mu           sync.Mutex
}

func newDeltaProcessor(baseState *api.FeatureResponse) *deltaProcessor {

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

	deltaProcessor := &deltaProcessor{
		featureState: atomic.Value{},
	}

	deltaProcessor.featureState.Store(featureState)
	return deltaProcessor
}

// It's hard to make this both concurrency safe and atomic for both readers and writes. To build our new state we need
// to hold a lock over the entire process to prevent internal races setting the final state out of order.
// However, once we have built the new state we can atomically swap it. This gives us mutex locked writes but
// lock free reads via snapshot()
func (dp *deltaProcessor) process(delta *api.ClientFeaturesDelta) (*api.FeatureResponse, error) {
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

	for _, event := range delta.Events {
		switch e := event.(type) {

		case *api.FeatureUpdatedEvent:
			featMap[e.Feature.Name] = &e.Feature

		case *api.FeatureRemovedEvent:
			delete(featMap, e.FeatureName)

		case *api.SegmentUpdatedEvent:
			segMap[e.Segment.Id] = e.Segment.Constraints

		case *api.SegmentRemovedEvent:
			delete(segMap, e.SegmentId)

		case *api.HydrationEvent:
			featMap = make(map[string]*api.Feature, len(e.Features))
			for _, f := range e.Features {
				featMap[f.Name] = &f
			}

			segMap = make(map[int][]api.Constraint, len(e.Segments))
			for _, seg := range e.Segments {
				segMap[seg.Id] = seg.Constraints
			}

		default:
			// Unknown event type - log but don't fail
			// This allows forward compatibility with new event types
		}
	}

	newState := &FeatureMemoryState{
		Features: featMap,
		Segments: segMap,
	}

	dp.featureState.Store(newState)

	return newState.asApiResponse(), nil
}

func (dp *deltaProcessor) snapshot() *FeatureMemoryState {
	v := dp.featureState.Load()
	return v.(*FeatureMemoryState)
}
