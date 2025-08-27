package unleash

import (
	"encoding/json"
	"testing"

	"github.com/Unleash/unleash-go-sdk/v5/api"
)

// TestStreamingDeltaIntegration tests the full delta processing flow
func TestStreamingDeltaIntegration(t *testing.T) {
	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")
	
	repo := &repository{
		segments: make(map[int][]api.Constraint),
	}
	
	channels := repositoryChannels{
		ready:  make(chan bool, 1),
		update: make(chan bool, 1),
		errorChannels: errorChannels{
			errors:   make(chan error, 10),
			warnings: make(chan error, 10),
		},
	}
	
	processor := newStreamingProcessor(storage, channels)
	
	t.Run("process initial hydration", func(t *testing.T) {
		hydrationJSON := `{
			"events": [{
				"type": "hydration",
				"eventId": 1,
				"features": [
					{"name": "feature-1", "enabled": true, "strategies": []},
					{"name": "feature-2", "enabled": false, "strategies": []}
				],
				"segments": [
					{"id": 1, "constraints": [{"contextName": "userId", "operator": "IN", "values": ["123"]}]},
					{"id": 2, "constraints": []}
				]
			}]
		}`
		
		var delta api.ClientFeaturesDelta
		err := json.Unmarshal([]byte(hydrationJSON), &delta)
		if err != nil {
			t.Fatalf("Failed to unmarshal hydration: %v", err)
		}
		
		err = processor.processDelta(&delta)
		if err != nil {
			t.Fatalf("Failed to process hydration: %v", err)
		}
		
		if feature, exists := storage.Get("feature-1"); !exists {
			t.Error("feature-1 should exist after hydration")
		} else if f, ok := feature.(api.Feature); ok && !f.Enabled {
			t.Error("feature-1 should be enabled")
		}
		
		if feature, exists := storage.Get("feature-2"); !exists {
			t.Error("feature-2 should exist after hydration")
		} else if f, ok := feature.(api.Feature); ok && f.Enabled {
			t.Error("feature-2 should be disabled")
		}
		
		segments := repo.segments
		if len(segments) != 2 {
			t.Errorf("Expected 2 segments, got %d", len(segments))
		}
		
		select {
		case <-channels.ready:
		default:
			t.Error("Expected ready signal after hydration")
		}
	})
	
	t.Run("process incremental updates", func(t *testing.T) {
		updateJSON := `{
			"events": [
				{
					"type": "feature-updated",
					"eventId": 2,
					"feature": {"name": "feature-1", "enabled": false, "strategies": []}
				},
				{
					"type": "feature-removed",
					"eventId": 3,
					"featureName": "feature-2",
					"project": "default"
				},
				{
					"type": "feature-updated",
					"eventId": 4,
					"feature": {"name": "feature-3", "enabled": true, "strategies": []}
				},
				{
					"type": "segment-removed",
					"eventId": 5,
					"segmentId": 1
				},
				{
					"type": "segment-updated",
					"eventId": 6,
					"segment": {"id": 3, "constraints": [{"contextName": "environment", "operator": "IN", "values": ["prod"]}]}
				}
			]
		}`
		
		var delta api.ClientFeaturesDelta
		err := json.Unmarshal([]byte(updateJSON), &delta)
		if err != nil {
			t.Fatalf("Failed to unmarshal update: %v", err)
		}
		
		err = processor.processDelta(&delta)
		if err != nil {
			t.Fatalf("Failed to process update: %v", err)
		}
		
		if feature, exists := storage.Get("feature-1"); !exists {
			t.Error("feature-1 should still exist")
		} else if f, ok := feature.(api.Feature); ok && f.Enabled {
			t.Error("feature-1 should be disabled after update")
		}
		
		if _, exists := storage.Get("feature-2"); exists {
			t.Error("feature-2 should not exist after removal")
		}
		
		if feature, exists := storage.Get("feature-3"); !exists {
			t.Error("feature-3 should exist after update")
		} else if f, ok := feature.(api.Feature); ok && !f.Enabled {
			t.Error("feature-3 should be enabled")
		}
		
		segments := repo.segments
		if len(segments) != 2 {
			t.Errorf("Expected 2 segments after updates, got %d", len(segments))
		}
		
		if _, exists := segments[1]; exists {
			t.Error("Segment 1 should not exist after removal")
		}
		
		if _, exists := segments[3]; !exists {
			t.Error("Segment 3 should exist after update")
		}
		
		select {
		case <-channels.update:
		default:
			t.Error("Expected update signal after incremental changes")
		}
	})
	
	t.Run("fallback for non-delta storage", func(t *testing.T) {
		regularStorage := &DefaultStorage{}
		regularStorage.Init("/tmp", "test-app-regular")
		
		regularProcessor := newStreamingProcessor(regularStorage, channels)
		
		updateJSON := `{
			"events": [
				{
					"type": "feature-updated",
					"eventId": 7,
					"feature": {"name": "test-feature", "enabled": true, "strategies": []}
				}
			]
		}`
		
		var delta api.ClientFeaturesDelta
		err := json.Unmarshal([]byte(updateJSON), &delta)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		
		err = regularProcessor.processDelta(&delta)
		if err != nil {
			t.Fatalf("Failed to process with regular storage: %v", err)
		}
		
		if feature, exists := regularStorage.Get("test-feature"); !exists {
			t.Error("Feature should exist even with regular storage")
		} else if f, ok := feature.(api.Feature); ok && !f.Enabled {
			t.Error("Feature should be enabled")
		}
	})
}

// TestDeltaEventBackwardCompatibility verifies backward compatibility
func TestDeltaEventBackwardCompatibility(t *testing.T) {
	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")
	
	repo := &repository{
		segments: make(map[int][]api.Constraint),
	}
	
	channels := repositoryChannels{
		ready:  make(chan bool, 1),
		update: make(chan bool, 1),
		errorChannels: errorChannels{
			errors:   make(chan error, 10),
			warnings: make(chan error, 10),
		},
	}
	
	processor := newStreamingProcessor(storage, channels)
	
	legacyResponse := api.FeatureResponse{
		Features: []api.Feature{
			{Name: "legacy-feature", Enabled: true},
		},
		Segments: []api.Segment{
			{Id: 1, Constraints: []api.Constraint{}},
		},
	}
	
	err := processor.processFeatureResponse(legacyResponse)
	if err != nil {
		t.Fatalf("Failed to process legacy format: %v", err)
	}
	
	if _, exists := storage.Get("legacy-feature"); !exists {
		t.Error("Legacy feature should exist")
	}
	
	segments := repo.segments
	if len(segments) != 1 {
		t.Errorf("Expected 1 segment from legacy format, got %d", len(segments))
	}
}