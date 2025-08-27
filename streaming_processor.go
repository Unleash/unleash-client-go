package unleash

import (
	"fmt"
	"sync"

	"github.com/Unleash/unleash-go-sdk/v5/api"
)

// streamingProcessor handles processing of streaming events and updating the feature storage
type streamingProcessor struct {
	storage            Storage
	repository         *repository // Repository reference for segment manipulation
	mu                 sync.RWMutex
	repositoryChannels repositoryChannels
	isReady            bool
}

// newStreamingProcessor creates a new streaming processor
func newStreamingProcessor(storage Storage, repo *repository, channels repositoryChannels) *streamingProcessor {
	return &streamingProcessor{
		storage:            storage,
		repository:         repo,
		repositoryChannels: channels,
		isReady:            false,
	}
}

// processFeatureResponse processes a feature response from streaming events
func (sp *streamingProcessor) processFeatureResponse(response api.FeatureResponse) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	// Update segments in repository
	sp.repository.Lock()
	for segmentId, constraints := range response.SegmentsMap() {
		sp.repository.segments[segmentId] = constraints
	}
	sp.repository.Unlock()

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
	sp.mu.Lock()
	defer sp.mu.Unlock()
	
	// Get current features from storage
	currentFeatures := make(map[string]interface{})
	for _, f := range sp.storage.List() {
		if feature, ok := f.(api.Feature); ok {
			currentFeatures[feature.Name] = feature
		}
	}
	
	// Apply delta events to the current state
	for _, event := range delta.Events {
		switch e := event.(type) {
		case *api.FeatureUpdatedEvent:
			currentFeatures[e.Feature.Name] = e.Feature
			
		case *api.FeatureRemovedEvent:
			delete(currentFeatures, e.FeatureName)
			
		case *api.SegmentUpdatedEvent:
			// Manipulate repository segments directly
			sp.repository.Lock()
			sp.repository.segments[e.Segment.Id] = e.Segment.Constraints
			sp.repository.Unlock()
			
		case *api.SegmentRemovedEvent:
			// Manipulate repository segments directly
			sp.repository.Lock()
			delete(sp.repository.segments, e.SegmentId)
			sp.repository.Unlock()
			
		case *api.HydrationEvent:
			// Replace entire state
			currentFeatures = make(map[string]interface{})
			for _, feature := range e.Features {
				currentFeatures[feature.Name] = feature
			}
			
			// Replace segments in repository
			sp.repository.Lock()
			sp.repository.segments = make(map[int][]api.Constraint)
			for _, segment := range e.Segments {
				sp.repository.segments[segment.Id] = segment.Constraints
			}
			sp.repository.Unlock()
			
		default:
			// Unknown event type - log but don't fail
			// This allows forward compatibility with new event types
		}
	}
	
	// Reset storage with the updated features
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

