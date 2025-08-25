package unleash

import (
	"fmt"
	"sync"

	"github.com/Unleash/unleash-go-sdk/v5/api"
)

// streamingProcessor handles processing of streaming events and updating the feature storage
type streamingProcessor struct {
	storage            Storage
	deltaStorage       DeltaStorage // Optional: only set if storage supports delta
	mu                 sync.RWMutex
	repositoryChannels repositoryChannels
	isReady            bool
}

// newStreamingProcessor creates a new streaming processor
func newStreamingProcessor(storage Storage, channels repositoryChannels) *streamingProcessor {
	sp := &streamingProcessor{
		storage:            storage,
		repositoryChannels: channels,
		isReady:            false,
	}
	
	// Check if storage supports delta operations
	if ds, ok := storage.(DeltaStorage); ok {
		sp.deltaStorage = ds
	}
	
	return sp
}

// processFeatureResponse processes a feature response from streaming events
func (sp *streamingProcessor) processFeatureResponse(response api.FeatureResponse) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Update segments in DeltaStorage if available
	if sp.deltaStorage != nil {
		for segmentId, constraints := range response.SegmentsMap() {
			if err := sp.deltaStorage.UpdateSegment(segmentId, constraints); err != nil {
				return fmt.Errorf("failed to update segment %d: %w", segmentId, err)
			}
		}
	}

	// Update storage with new features
	sp.storage.Reset(response.FeatureMap(), true)

	// Signal ready or update
	if !sp.isReady {
		sp.isReady = true
		select {
		case sp.repositoryChannels.ready <- true:
		default:
		}
	} else {
		select {
		case sp.repositoryChannels.update <- true:
		default:
		}
	}

	return nil
}


// processDelta processes a delta update from streaming events
func (sp *streamingProcessor) processDelta(delta *api.ClientFeaturesDelta) error {
	if sp.deltaStorage == nil {
		// Fallback: convert delta to full update if storage doesn't support delta
		return sp.processDeltaWithoutDeltaStorage(delta)
	}
	
	sp.mu.Lock()
	defer sp.mu.Unlock()
	
	for _, event := range delta.Events {
		switch e := event.(type) {
		case *api.FeatureUpdatedEvent:
			if err := sp.deltaStorage.Update(e.Feature.Name, e.Feature); err != nil {
				return fmt.Errorf("failed to update feature %s: %w", e.Feature.Name, err)
			}
			
		case *api.FeatureRemovedEvent:
			if err := sp.deltaStorage.Delete(e.FeatureName); err != nil {
				return fmt.Errorf("failed to remove feature %s: %w", e.FeatureName, err)
			}
			
		case *api.SegmentUpdatedEvent:
			if err := sp.deltaStorage.UpdateSegment(e.Segment.Id, e.Segment.Constraints); err != nil {
				return fmt.Errorf("failed to update segment %d: %w", e.Segment.Id, err)
			}
			
		case *api.SegmentRemovedEvent:
			if err := sp.deltaStorage.DeleteSegment(e.SegmentId); err != nil {
				return fmt.Errorf("failed to remove segment %d: %w", e.SegmentId, err)
			}
			
		case *api.HydrationEvent:
			// Clear existing state and replace with hydration data
			newData := make(map[string]interface{})
			for _, feature := range e.Features {
				newData[feature.Name] = feature
			}
			
			// Reset the storage with new data
			if err := sp.storage.Reset(newData, false); err != nil {
				return fmt.Errorf("failed to reset storage during hydration: %w", err)
			}
			
			// Clear existing segments and add new ones
			// First clear all existing segments
			if currentSegments := sp.deltaStorage.GetSegments(); len(currentSegments) > 0 {
				for segmentId := range currentSegments {
					if err := sp.deltaStorage.DeleteSegment(segmentId); err != nil {
						return fmt.Errorf("failed to clear segment %d during hydration: %w", segmentId, err)
					}
				}
			}
			
			// Add new segments
			for _, segment := range e.Segments {
				if err := sp.deltaStorage.UpdateSegment(segment.Id, segment.Constraints); err != nil {
					return fmt.Errorf("failed to hydrate segment %d: %w", segment.Id, err)
				}
			}
			
			// Persist after hydration
			if err := sp.storage.Persist(); err != nil {
				return fmt.Errorf("failed to persist after hydration: %w", err)
			}
			
		default:
			// Unknown event type - log but don't fail
			// This allows forward compatibility with new event types
		}
	}
	
	// Signal ready or update
	if !sp.isReady {
		sp.isReady = true
		select {
		case sp.repositoryChannels.ready <- true:
		default:
		}
	} else {
		select {
		case sp.repositoryChannels.update <- true:
		default:
		}
	}
	
	return nil
}

// processDeltaWithoutDeltaStorage handles delta events when storage doesn't support DeltaStorage
func (sp *streamingProcessor) processDeltaWithoutDeltaStorage(delta *api.ClientFeaturesDelta) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	
	// Get current features from storage
	currentFeatures := make(map[string]interface{})
	for _, f := range sp.storage.List() {
		if feature, ok := f.(api.Feature); ok {
			currentFeatures[feature.Name] = feature
		}
	}
	
	// Note: Without DeltaStorage, we can't properly handle segments in delta mode
	// This fallback only handles features. Segments would need to be handled differently
	// if we need to support non-DeltaStorage in streaming mode (which shouldn't happen
	// as streaming requires delta support)
	
	// Apply delta events to the current state
	for _, event := range delta.Events {
		switch e := event.(type) {
		case *api.FeatureUpdatedEvent:
			currentFeatures[e.Feature.Name] = e.Feature
			
		case *api.FeatureRemovedEvent:
			delete(currentFeatures, e.FeatureName)
			
		case *api.SegmentUpdatedEvent:
			// Cannot handle segments without DeltaStorage
			// Log warning or return error
			return fmt.Errorf("cannot handle segment updates without DeltaStorage support")
			
		case *api.SegmentRemovedEvent:
			// Cannot handle segments without DeltaStorage
			return fmt.Errorf("cannot handle segment removal without DeltaStorage support")
			
		case *api.HydrationEvent:
			// Replace entire state
			currentFeatures = make(map[string]interface{})
			for _, feature := range e.Features {
				currentFeatures[feature.Name] = feature
			}
			
			// Cannot handle segments without DeltaStorage
			if len(e.Segments) > 0 {
				return fmt.Errorf("cannot handle segments in hydration without DeltaStorage support")
			}
		}
	}
	
	// Reset storage with the updated state
	if err := sp.storage.Reset(currentFeatures, true); err != nil {
		return fmt.Errorf("failed to reset storage after delta: %w", err)
	}
	
	// Signal ready or update
	if !sp.isReady {
		sp.isReady = true
		select {
		case sp.repositoryChannels.ready <- true:
		default:
		}
	} else {
		select {
		case sp.repositoryChannels.update <- true:
		default:
		}
	}
	
	return nil
}