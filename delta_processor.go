package unleash

import (
	"fmt"
	"sync"

	"github.com/Unleash/unleash-go-sdk/v5/api"
)

type deltaProcessor struct {
	repository         *repository // ideally this shouldn't be necessary, but for now we're going going to use it to resolve a snapshot of the feature state
	mu                 sync.RWMutex
	repositoryChannels repositoryChannels
	isReady            bool
}

func newDeltaProcessor(repo *repository, channels repositoryChannels) *deltaProcessor {
	return &deltaProcessor{
		repository:         repo,
		repositoryChannels: channels,
		isReady:            false,
	}
}

// process processes a delta update from streaming or API events
// this needs some love, currently it's too intertwined with the repository
func (dp *deltaProcessor) process(delta *api.ClientFeaturesDelta) error {
	if delta == nil {
		return fmt.Errorf("delta is nil")
	}

	dp.mu.Lock()
	defer dp.mu.Unlock()

	snap := dp.repository.snapshot()

	featMap := make(map[string]api.Feature, len(snap.Features))
	for name, f := range snap.Features {
		featMap[name] = *f
	}

	segMap := make(map[int][]api.Constraint, len(snap.Segments))
	for id, constraints := range snap.Segments {
		segMap[id] = constraints
	}

	// Apply deltas
	for _, event := range delta.Events {
		switch e := event.(type) {

		case *api.FeatureUpdatedEvent:
			featMap[e.Feature.Name] = e.Feature

		case *api.FeatureRemovedEvent:
			delete(featMap, e.FeatureName)

		case *api.SegmentUpdatedEvent:
			segMap[e.Segment.Id] = e.Segment.Constraints

		case *api.SegmentRemovedEvent:
			delete(segMap, e.SegmentId)

		case *api.HydrationEvent:
			featMap = make(map[string]api.Feature, len(e.Features))
			for _, f := range e.Features {
				featMap[f.Name] = f
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

	newFeatureList := make([]api.Feature, 0, len(featMap))
	for _, feature := range featMap {
		newFeatureList = append(newFeatureList, feature)
	}

	newSegmentList := make([]api.Segment, 0, len(segMap))
	for id, c := range segMap {
		newSegmentList = append(newSegmentList, api.Segment{
			Id:          id,
			Constraints: c,
		})
	}

	state := &api.FeatureResponse{
		Features: newFeatureList,
		Segments: newSegmentList,
	}

	if err := dp.repository.saveState(state); err != nil {
		return fmt.Errorf("failed to persist after delta: %w", err)
	}

	if !dp.isReady {
		dp.isReady = true
		select {
		case dp.repositoryChannels.ready <- true:
		default:
		}
	} else {
		select {
		case dp.repositoryChannels.update <- true:
		default:
		}
	}

	return nil
}
