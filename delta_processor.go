package unleash

import (
	"fmt"
	"sync"

	"github.com/Unleash/unleash-go-sdk/v5/api"
)

type deltaProcessor struct {
	storage            Storage
	repository         *repository // Repository reference for segment manipulation
	mu                 sync.RWMutex
	repositoryChannels repositoryChannels
	isReady            bool
}

func newDeltaProcessor(storage Storage, repo *repository, channels repositoryChannels) *deltaProcessor {
	return &deltaProcessor{
		storage:            storage,
		repository:         repo,
		repositoryChannels: channels,
		isReady:            false,
	}
}

// process processes a delta update from streaming or API events
func (dp *deltaProcessor) process(delta *api.ClientFeaturesDelta) error {
	if delta == nil {
		return fmt.Errorf("delta is nil")
	}

	dp.mu.Lock()
	defer dp.mu.Unlock()

	currentFeatures := make(map[string]interface{})
	for _, f := range dp.storage.List() {
		if feature, ok := f.(api.Feature); ok {
			currentFeatures[feature.Name] = feature
		}
	}

	segments := make(map[int][]api.Constraint)
	dp.repository.RLock()
	for id, constraints := range dp.repository.segments {
		segments[id] = constraints
	}
	dp.repository.RUnlock()

	// Apply delta events to the current state
	for _, event := range delta.Events {
		switch e := event.(type) {
		case *api.FeatureUpdatedEvent:
			currentFeatures[e.Feature.Name] = e.Feature

		case *api.FeatureRemovedEvent:
			delete(currentFeatures, e.FeatureName)

		case *api.SegmentUpdatedEvent:
			segments[e.Segment.Id] = e.Segment.Constraints

		case *api.SegmentRemovedEvent:
			delete(segments, e.SegmentId)

		case *api.HydrationEvent:
			// Replace entire state
			currentFeatures = make(map[string]interface{})
			for _, feature := range e.Features {
				currentFeatures[feature.Name] = feature
			}

			// Replace segments
			segments = make(map[int][]api.Constraint)
			for _, segment := range e.Segments {
				segments[segment.Id] = segment.Constraints
			}

		default:
			// Unknown event type - log but don't fail
			// This allows forward compatibility with new event types
		}
	}

	if err := dp.repository.updateStorageWithDelta(currentFeatures, segments); err != nil {
		return fmt.Errorf("failed to reset storage after delta: %w", err)
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
