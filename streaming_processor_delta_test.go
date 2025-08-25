package unleash

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5/api"
)

// MockStreamingProcessor for testing delta processing
type MockStreamingProcessor struct {
	storage            DeltaStorage
	mu                 sync.RWMutex
	repositoryChannels repositoryChannels
	isReady            bool
	processedEvents    []string
	lastEventId        int
}

func NewMockStreamingProcessor(storage DeltaStorage) *MockStreamingProcessor {
	return &MockStreamingProcessor{
		storage:         storage,
		processedEvents: []string{},
		repositoryChannels: repositoryChannels{
			ready:  make(chan bool, 1),
			update: make(chan bool, 1),
		},
	}
}

func (p *MockStreamingProcessor) processDelta(delta *api.ClientFeaturesDelta) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for _, event := range delta.Events {
		// Track processed events for testing
		p.processedEvents = append(p.processedEvents, event.GetType())
		p.lastEventId = event.GetEventId()
		
		switch e := event.(type) {
		case *api.FeatureUpdatedEvent:
			if err := p.storage.Update(e.Feature.Name, e.Feature); err != nil {
				return fmt.Errorf("failed to update feature %s: %w", e.Feature.Name, err)
			}
			
		case *api.FeatureRemovedEvent:
			if err := p.storage.Delete(e.FeatureName); err != nil {
				return fmt.Errorf("failed to remove feature %s: %w", e.FeatureName, err)
			}
			
		case *api.SegmentUpdatedEvent:
			if err := p.storage.UpdateSegment(e.Segment.Id, e.Segment.Constraints); err != nil {
				return fmt.Errorf("failed to update segment %d: %w", e.Segment.Id, err)
			}
			
		case *api.SegmentRemovedEvent:
			if err := p.storage.DeleteSegment(e.SegmentId); err != nil {
				return fmt.Errorf("failed to remove segment %d: %w", e.SegmentId, err)
			}
			
		case *api.HydrationEvent:
			// Clear existing state
			p.storage.Reset(make(map[string]interface{}), false)
			
			// Clear all existing segments in storage
			if deltaStorage, ok := p.storage.(DeltaStorage); ok {
				// Get all current segments and delete them
				currentSegments := deltaStorage.GetSegments()
				for segmentId := range currentSegments {
					deltaStorage.DeleteSegment(segmentId)
				}
			}
			
			// Apply all features
			for _, feature := range e.Features {
				if err := p.storage.Update(feature.Name, feature); err != nil {
					return fmt.Errorf("failed to hydrate feature %s: %w", feature.Name, err)
				}
			}
			
			// Apply all segments
			for _, segment := range e.Segments {
				if err := p.storage.UpdateSegment(segment.Id, segment.Constraints); err != nil {
					return fmt.Errorf("failed to hydrate segment %d: %w", segment.Id, err)
				}
			}
			
		default:
			// Unknown event type - log but don't fail
			p.processedEvents = append(p.processedEvents, "unknown")
		}
	}
	
	// Signal ready or update
	if !p.isReady {
		p.isReady = true
		select {
		case p.repositoryChannels.ready <- true:
		default:
		}
	} else {
		select {
		case p.repositoryChannels.update <- true:
		default:
		}
	}
	
	return nil
}

func TestProcessDeltaSingleEvent(t *testing.T) {
	storage := NewMockDeltaStorage()
	processor := NewMockStreamingProcessor(storage)
	
	t.Run("feature-updated", func(t *testing.T) {
		delta := &api.ClientFeaturesDelta{
			Events: []api.DeltaEvent{
				&api.FeatureUpdatedEvent{
					Type:    "feature-updated",
					EventId: 1,
					Feature: api.Feature{
						Name:    "test-feature",
						Enabled: true,
					},
				},
			},
		}
		
		err := processor.processDelta(delta)
		if err != nil {
			t.Errorf("processDelta() error = %v", err)
		}
		
		// Verify feature was added to storage
		feature, exists := storage.Get("test-feature")
		if !exists {
			t.Error("Feature should exist in storage")
		}
		
		if f, ok := feature.(api.Feature); ok {
			if !f.Enabled {
				t.Error("Feature should be enabled")
			}
		}
		
		// Verify event was tracked
		if len(processor.processedEvents) != 1 || processor.processedEvents[0] != "feature-updated" {
			t.Error("Event not tracked correctly")
		}
		
		// Verify ready signal was sent
		select {
		case <-processor.repositoryChannels.ready:
			// Good
		default:
			t.Error("Ready signal should have been sent")
		}
	})
	
	t.Run("feature-removed", func(t *testing.T) {
		// Add a feature first
		storage.Update("remove-me", api.Feature{Name: "remove-me"})
		
		delta := &api.ClientFeaturesDelta{
			Events: []api.DeltaEvent{
				&api.FeatureRemovedEvent{
					Type:        "feature-removed",
					EventId:     2,
					FeatureName: "remove-me",
					Project:     "default",
				},
			},
		}
		
		err := processor.processDelta(delta)
		if err != nil {
			t.Errorf("processDelta() error = %v", err)
		}
		
		// Verify feature was removed from storage
		_, exists := storage.Get("remove-me")
		if exists {
			t.Error("Feature should not exist in storage")
		}
	})
	
	t.Run("segment-updated", func(t *testing.T) {
		delta := &api.ClientFeaturesDelta{
			Events: []api.DeltaEvent{
				&api.SegmentUpdatedEvent{
					Type:    "segment-updated",
					EventId: 3,
					Segment: api.Segment{
						Id: 1,
						Constraints: []api.Constraint{
							{
								ContextName: "userId",
								Operator:    "IN",
								Values:      []string{"123"},
							},
						},
					},
				},
			},
		}
		
		err := processor.processDelta(delta)
		if err != nil {
			t.Errorf("processDelta() error = %v", err)
		}
		
		// Verify segment was added
		segments := storage.GetSegments()
		if len(segments) != 1 {
			t.Errorf("Expected 1 segment, got %d", len(segments))
		}
		
		if len(segments[1]) != 1 {
			t.Error("Segment constraints not stored correctly")
		}
	})
	
	t.Run("segment-removed", func(t *testing.T) {
		// Add a segment first
		storage.UpdateSegment(2, []api.Constraint{{ContextName: "test"}})
		
		delta := &api.ClientFeaturesDelta{
			Events: []api.DeltaEvent{
				&api.SegmentRemovedEvent{
					Type:      "segment-removed",
					EventId:   4,
					SegmentId: 2,
				},
			},
		}
		
		err := processor.processDelta(delta)
		if err != nil {
			t.Errorf("processDelta() error = %v", err)
		}
		
		// Verify segment was removed
		segments := storage.GetSegments()
		if _, exists := segments[2]; exists {
			t.Error("Segment should not exist after removal")
		}
	})
}

func TestProcessDeltaHydration(t *testing.T) {
	storage := NewMockDeltaStorage()
	processor := NewMockStreamingProcessor(storage)
	
	// Add some existing data
	storage.Update("old-feature", api.Feature{Name: "old-feature"})
	storage.UpdateSegment(99, []api.Constraint{{ContextName: "old"}})
	
	// Hydration event should replace everything
	delta := &api.ClientFeaturesDelta{
		Events: []api.DeltaEvent{
			&api.HydrationEvent{
				Type:    "hydration",
				EventId: 5,
				Features: []api.Feature{
					{Name: "new-feature-1", Enabled: true},
					{Name: "new-feature-2", Enabled: false},
				},
				Segments: []api.Segment{
					{Id: 1, Constraints: []api.Constraint{{ContextName: "new"}}},
					{Id: 2, Constraints: []api.Constraint{}},
				},
			},
		},
	}
	
	err := processor.processDelta(delta)
	if err != nil {
		t.Errorf("processDelta() error = %v", err)
	}
	
	// Old feature should be gone
	if _, exists := storage.Get("old-feature"); exists {
		t.Error("Old feature should not exist after hydration")
	}
	
	// New features should exist
	if _, exists := storage.Get("new-feature-1"); !exists {
		t.Error("new-feature-1 should exist")
	}
	if _, exists := storage.Get("new-feature-2"); !exists {
		t.Error("new-feature-2 should exist")
	}
	
	// Old segment should be gone, new segments should exist
	segments := storage.GetSegments()
	if len(segments) != 2 {
		t.Errorf("Expected 2 segments after hydration, got %d", len(segments))
	}
	if _, exists := segments[99]; exists {
		t.Error("Old segment should not exist after hydration")
	}
	if _, exists := segments[1]; !exists {
		t.Error("Segment 1 should exist")
	}
	if _, exists := segments[2]; !exists {
		t.Error("Segment 2 should exist")
	}
}

func TestProcessDeltaMultipleEvents(t *testing.T) {
	storage := NewMockDeltaStorage()
	processor := NewMockStreamingProcessor(storage)
	
	// Process multiple events in sequence
	delta := &api.ClientFeaturesDelta{
		Events: []api.DeltaEvent{
			&api.FeatureUpdatedEvent{
				Type:    "feature-updated",
				EventId: 1,
				Feature: api.Feature{Name: "feature-1", Enabled: true},
			},
			&api.FeatureUpdatedEvent{
				Type:    "feature-updated",
				EventId: 2,
				Feature: api.Feature{Name: "feature-2", Enabled: false},
			},
			&api.SegmentUpdatedEvent{
				Type:    "segment-updated",
				EventId: 3,
				Segment: api.Segment{Id: 1, Constraints: []api.Constraint{}},
			},
			&api.FeatureRemovedEvent{
				Type:        "feature-removed",
				EventId:     4,
				FeatureName: "feature-1",
				Project:     "default",
			},
			&api.SegmentRemovedEvent{
				Type:      "segment-removed",
				EventId:   5,
				SegmentId: 1,
			},
		},
	}
	
	err := processor.processDelta(delta)
	if err != nil {
		t.Errorf("processDelta() error = %v", err)
	}
	
	// Verify final state
	if _, exists := storage.Get("feature-1"); exists {
		t.Error("feature-1 should not exist (was removed)")
	}
	if _, exists := storage.Get("feature-2"); !exists {
		t.Error("feature-2 should exist")
	}
	
	segments := storage.GetSegments()
	if len(segments) != 0 {
		t.Error("No segments should exist (was removed)")
	}
	
	// Verify all events were processed in order
	expectedEvents := []string{
		"feature-updated",
		"feature-updated",
		"segment-updated",
		"feature-removed",
		"segment-removed",
	}
	
	if len(processor.processedEvents) != len(expectedEvents) {
		t.Errorf("Expected %d events, got %d", len(expectedEvents), len(processor.processedEvents))
	}
	
	for i, expected := range expectedEvents {
		if processor.processedEvents[i] != expected {
			t.Errorf("Event %d: got %s, want %s", i, processor.processedEvents[i], expected)
		}
	}
	
	// Verify last event ID
	if processor.lastEventId != 5 {
		t.Errorf("Last event ID = %d, want 5", processor.lastEventId)
	}
}

func TestProcessDeltaEventOrdering(t *testing.T) {
	storage := NewMockDeltaStorage()
	processor := NewMockStreamingProcessor(storage)
	
	// Events should be processed in the order they appear in the array,
	// regardless of their eventId values
	delta := &api.ClientFeaturesDelta{
		Events: []api.DeltaEvent{
			&api.FeatureUpdatedEvent{
				Type:    "feature-updated",
				EventId: 3,
				Feature: api.Feature{Name: "feature-A", Enabled: true},
			},
			&api.FeatureUpdatedEvent{
				Type:    "feature-updated",
				EventId: 1,
				Feature: api.Feature{Name: "feature-A", Enabled: false},
			},
			&api.FeatureUpdatedEvent{
				Type:    "feature-updated",
				EventId: 2,
				Feature: api.Feature{Name: "feature-A", Enabled: true},
			},
		},
	}
	
	err := processor.processDelta(delta)
	if err != nil {
		t.Errorf("processDelta() error = %v", err)
	}
	
	// The final state should reflect the last update in the array
	feature, exists := storage.Get("feature-A")
	if !exists {
		t.Fatal("feature-A should exist")
	}
	
	if f, ok := feature.(api.Feature); ok {
		if !f.Enabled {
			t.Error("feature-A should be enabled (last update in array)")
		}
	}
	
	// Last processed event ID should be from the last event in array
	if processor.lastEventId != 2 {
		t.Errorf("Last event ID = %d, want 2", processor.lastEventId)
	}
}

func TestProcessDeltaErrorHandling(t *testing.T) {
	t.Run("storage error", func(t *testing.T) {
		storage := NewMockDeltaStorage()
		storage.failPersist = true
		processor := NewMockStreamingProcessor(storage)
		
		delta := &api.ClientFeaturesDelta{
			Events: []api.DeltaEvent{
				&api.FeatureUpdatedEvent{
					Type:    "feature-updated",
					EventId: 1,
					Feature: api.Feature{Name: "test", Enabled: true},
				},
			},
		}
		
		err := processor.processDelta(delta)
		if err == nil {
			t.Error("Expected error from storage failure")
		}
	})
	
	t.Run("unknown event type", func(t *testing.T) {
		storage := NewMockDeltaStorage()
		_ = NewMockStreamingProcessor(storage)
		
		// This would require a custom event type that implements DeltaEvent
		// For now, we track "unknown" in processedEvents when type assertion fails
		// The actual implementation would need to handle this gracefully
	})
}

func TestProcessDeltaReadyAndUpdateSignals(t *testing.T) {
	storage := NewMockDeltaStorage()
	processor := NewMockStreamingProcessor(storage)
	
	// First delta should send ready signal
	delta1 := &api.ClientFeaturesDelta{
		Events: []api.DeltaEvent{
			&api.FeatureUpdatedEvent{
				Type:    "feature-updated",
				EventId: 1,
				Feature: api.Feature{Name: "feature-1", Enabled: true},
			},
		},
	}
	
	err := processor.processDelta(delta1)
	if err != nil {
		t.Errorf("processDelta() error = %v", err)
	}
	
	// Should receive ready signal
	select {
	case <-processor.repositoryChannels.ready:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("Should have received ready signal")
	}
	
	// Subsequent deltas should send update signal
	delta2 := &api.ClientFeaturesDelta{
		Events: []api.DeltaEvent{
			&api.FeatureUpdatedEvent{
				Type:    "feature-updated",
				EventId: 2,
				Feature: api.Feature{Name: "feature-2", Enabled: true},
			},
		},
	}
	
	err = processor.processDelta(delta2)
	if err != nil {
		t.Errorf("processDelta() error = %v", err)
	}
	
	// Should receive update signal
	select {
	case <-processor.repositoryChannels.update:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("Should have received update signal")
	}
}

func TestProcessDeltaConcurrency(t *testing.T) {
	storage := NewMockDeltaStorage()
	processor := NewMockStreamingProcessor(storage)
	
	// Process multiple deltas concurrently
	var wg sync.WaitGroup
	numDeltas := 10
	wg.Add(numDeltas)
	
	for i := 0; i < numDeltas; i++ {
		go func(idx int) {
			defer wg.Done()
			
			delta := &api.ClientFeaturesDelta{
				Events: []api.DeltaEvent{
					&api.FeatureUpdatedEvent{
						Type:    "feature-updated",
						EventId: idx,
						Feature: api.Feature{
							Name:    fmt.Sprintf("feature-%d", idx),
							Enabled: true,
						},
					},
				},
			}
			
			processor.processDelta(delta)
		}(i)
	}
	
	// Wait for all deltas to be processed
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()
	
	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Concurrent processing timed out")
	}
	
	// Verify all features were added
	for i := 0; i < numDeltas; i++ {
		featureName := fmt.Sprintf("feature-%d", i)
		if _, exists := storage.Get(featureName); !exists {
			t.Errorf("Feature %s should exist", featureName)
		}
	}
}

func TestProcessDeltaIdempotency(t *testing.T) {
	storage := NewMockDeltaStorage()
	processor := NewMockStreamingProcessor(storage)
	
	// Process the same delta multiple times
	delta := &api.ClientFeaturesDelta{
		Events: []api.DeltaEvent{
			&api.FeatureUpdatedEvent{
				Type:    "feature-updated",
				EventId: 1,
				Feature: api.Feature{
					Name:        "idempotent-feature",
					Enabled:     true,
					Description: "Test idempotency",
				},
			},
		},
	}
	
	// Process the delta three times
	for i := 0; i < 3; i++ {
		err := processor.processDelta(delta)
		if err != nil {
			t.Errorf("processDelta() attempt %d error = %v", i, err)
		}
	}
	
	// Feature should exist with correct state
	feature, exists := storage.Get("idempotent-feature")
	if !exists {
		t.Fatal("Feature should exist")
	}
	
	if f, ok := feature.(api.Feature); ok {
		if !f.Enabled || f.Description != "Test idempotency" {
			t.Error("Feature state incorrect after multiple identical updates")
		}
	}
	
	// Storage should have been called 3 times (no deduplication in this implementation)
	if storage.updateCallCount != 3 {
		t.Errorf("Update called %d times, want 3", storage.updateCallCount)
	}
}

func TestProcessDeltaEmptyEvents(t *testing.T) {
	storage := NewMockDeltaStorage()
	processor := NewMockStreamingProcessor(storage)
	
	// Empty delta
	delta := &api.ClientFeaturesDelta{
		Events: []api.DeltaEvent{},
	}
	
	err := processor.processDelta(delta)
	if err != nil {
		t.Errorf("processDelta() with empty events error = %v", err)
	}
	
	// No events should have been processed
	if len(processor.processedEvents) != 0 {
		t.Error("No events should have been processed")
	}
}