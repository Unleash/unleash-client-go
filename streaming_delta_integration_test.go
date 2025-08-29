package unleash

import (
	"encoding/json"
	"testing"

	"github.com/Unleash/unleash-go-sdk/v5/api"
	"github.com/stretchr/testify/assert"
)

// TestStreamingDeltaIntegration tests the full delta processing flow
func TestStreamingDeltaIntegration(t *testing.T) {
	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")
	
	options := repositoryOptions{
		storage: storage,
	}
	
	repo := &repository{
		segments: make(map[int][]api.Constraint),
		options:  options,
	}
	
	channels := repositoryChannels{
		ready:  make(chan bool, 1),
		update: make(chan bool, 1),
		errorChannels: errorChannels{
			errors:   make(chan error, 10),
			warnings: make(chan error, 10),
		},
	}
	
	processor := newStreamingProcessor(storage, repo, channels)
	
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
}

// TestStreamingDelta_MultipleDeltaEvents tests processing multiple delta events in sequence
func TestStreamingDelta_MultipleDeltaEvents(t *testing.T) {
	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")
	
	options := repositoryOptions{
		storage: storage,
	}
	
	repo := &repository{
		segments: make(map[int][]api.Constraint),
		options:  options,
	}
	
	channels := repositoryChannels{
		ready:  make(chan bool, 1),
		update: make(chan bool, 1),
		errorChannels: errorChannels{
			errors:   make(chan error, 10),
			warnings: make(chan error, 10),
		},
	}
	
	processor := newStreamingProcessor(storage, repo, channels)
	
	// Initial hydration with two features
	hydrationJSON := `{
		"events": [{
			"type": "hydration",
			"eventId": 1,
			"features": [
				{"name": "feature-a", "enabled": true, "strategies": [{"name": "default"}]},
				{"name": "feature-b", "enabled": true, "strategies": [{"name": "default"}]}
			],
			"segments": []
		}]
	}`
	
	var hydrationDelta api.ClientFeaturesDelta
	err := json.Unmarshal([]byte(hydrationJSON), &hydrationDelta)
	assert.NoError(t, err)
	
	err = processor.processDelta(&hydrationDelta)
	assert.NoError(t, err)
	
	// Verify initial state
	if feature, exists := storage.Get("feature-a"); !exists {
		t.Error("feature-a should exist after hydration")
	} else if f, ok := feature.(api.Feature); ok && !f.Enabled {
		t.Error("feature-a should be enabled")
	}
	
	if feature, exists := storage.Get("feature-b"); !exists {
		t.Error("feature-b should exist after hydration")
	} else if f, ok := feature.(api.Feature); ok && !f.Enabled {
		t.Error("feature-b should be enabled")
	}
	
	// Process multiple updates
	updatesJSON := `{
		"events": [
			{
				"type": "feature-updated",
				"eventId": 2,
				"feature": {"name": "feature-a", "enabled": false, "strategies": [{"name": "default"}]}
			},
			{
				"type": "feature-updated",
				"eventId": 3,
				"feature": {"name": "feature-c", "enabled": true, "strategies": [{"name": "default"}]}
			},
			{
				"type": "feature-removed",
				"eventId": 4,
				"featureName": "feature-b",
				"project": "default"
			}
		]
	}`
	
	var updatesDelta api.ClientFeaturesDelta
	err = json.Unmarshal([]byte(updatesJSON), &updatesDelta)
	assert.NoError(t, err)
	
	err = processor.processDelta(&updatesDelta)
	assert.NoError(t, err)
	
	// Verify final state
	if feature, exists := storage.Get("feature-a"); !exists {
		t.Error("feature-a should still exist")
	} else if f, ok := feature.(api.Feature); ok && f.Enabled {
		t.Error("feature-a should be disabled after update")
	}
	
	if _, exists := storage.Get("feature-b"); exists {
		t.Error("feature-b should not exist after removal")
	}
	
	if feature, exists := storage.Get("feature-c"); !exists {
		t.Error("feature-c should exist after update")
	} else if f, ok := feature.(api.Feature); ok && !f.Enabled {
		t.Error("feature-c should be enabled")
	}
}

// TestStreamingDelta_SegmentUpdates tests segment updates via delta events
func TestStreamingDelta_SegmentUpdates(t *testing.T) {
	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")
	
	options := repositoryOptions{
		storage: storage,
	}
	
	repo := &repository{
		segments: make(map[int][]api.Constraint),
		options:  options,
	}
	
	channels := repositoryChannels{
		ready:  make(chan bool, 1),
		update: make(chan bool, 1),
		errorChannels: errorChannels{
			errors:   make(chan error, 10),
			warnings: make(chan error, 10),
		},
	}
	
	processor := newStreamingProcessor(storage, repo, channels)
	
	// Hydration with feature using segments
	hydrationJSON := `{
		"events": [{
			"type": "hydration",
			"eventId": 1,
			"features": [
				{
					"name": "segmented-feature",
					"enabled": true,
					"strategies": [{
						"name": "default",
						"segments": [1]
					}]
				}
			],
			"segments": [
				{
					"id": 1,
					"constraints": [{
						"contextName": "userId",
						"operator": "IN",
						"values": ["123"]
					}]
				}
			]
		}]
	}`
	
	var hydrationDelta api.ClientFeaturesDelta
	err := json.Unmarshal([]byte(hydrationJSON), &hydrationDelta)
	assert.NoError(t, err)
	
	err = processor.processDelta(&hydrationDelta)
	assert.NoError(t, err)
	
	// Verify initial segment state
	if constraints, exists := repo.segments[1]; !exists {
		t.Error("Segment 1 should exist")
	} else if len(constraints) != 1 {
		t.Errorf("Segment 1 should have 1 constraint, got %d", len(constraints))
	}
	
	// Update segment to include more users
	segmentUpdateJSON := `{
		"events": [{
			"type": "segment-updated",
			"eventId": 2,
			"segment": {
				"id": 1,
				"constraints": [{
					"contextName": "userId",
					"operator": "IN",
					"values": ["123", "456"]
				}]
			}
		}]
	}`
	
	var segmentUpdateDelta api.ClientFeaturesDelta
	err = json.Unmarshal([]byte(segmentUpdateJSON), &segmentUpdateDelta)
	assert.NoError(t, err)
	
	err = processor.processDelta(&segmentUpdateDelta)
	assert.NoError(t, err)
	
	// Verify updated segment
	if constraints, exists := repo.segments[1]; !exists {
		t.Error("Segment 1 should still exist")
	} else if len(constraints) != 1 {
		t.Errorf("Segment 1 should have 1 constraint, got %d", len(constraints))
	} else if len(constraints[0].Values) != 2 {
		t.Errorf("Segment 1 constraint should have 2 values, got %d", len(constraints[0].Values))
	}
	
	// Add new segment and update feature to use both
	multiSegmentUpdateJSON := `{
		"events": [
			{
				"type": "segment-updated",
				"eventId": 3,
				"segment": {
					"id": 2,
					"constraints": [{
						"contextName": "environment",
						"operator": "IN",
						"values": ["production"]
					}]
				}
			},
			{
				"type": "feature-updated",
				"eventId": 4,
				"feature": {
					"name": "segmented-feature",
					"enabled": true,
					"strategies": [{
						"name": "default",
						"segments": [1, 2]
					}]
				}
			}
		]
	}`
	
	var multiSegmentDelta api.ClientFeaturesDelta
	err = json.Unmarshal([]byte(multiSegmentUpdateJSON), &multiSegmentDelta)
	assert.NoError(t, err)
	
	err = processor.processDelta(&multiSegmentDelta)
	assert.NoError(t, err)
	
	// Verify both segments exist
	if _, exists := repo.segments[1]; !exists {
		t.Error("Segment 1 should still exist")
	}
	if _, exists := repo.segments[2]; !exists {
		t.Error("Segment 2 should exist")
	}
	
	// Verify feature has been updated with both segments
	if feature, exists := storage.Get("segmented-feature"); !exists {
		t.Error("segmented-feature should exist")
	} else if f, ok := feature.(api.Feature); ok {
		if len(f.Strategies) != 1 {
			t.Errorf("Feature should have 1 strategy, got %d", len(f.Strategies))
		} else if len(f.Strategies[0].Segments) != 2 {
			t.Errorf("Strategy should have 2 segments, got %d", len(f.Strategies[0].Segments))
		}
	}
}

// TestStreamingDelta_ErrorHandling tests error handling in delta processing
func TestStreamingDelta_ErrorHandling(t *testing.T) {
	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")
	
	options := repositoryOptions{
		storage: storage,
	}
	
	repo := &repository{
		segments: make(map[int][]api.Constraint),
		options:  options,
	}
	
	channels := repositoryChannels{
		ready:  make(chan bool, 1),
		update: make(chan bool, 1),
		errorChannels: errorChannels{
			errors:   make(chan error, 10),
			warnings: make(chan error, 10),
		},
	}
	
	processor := newStreamingProcessor(storage, repo, channels)
	
	// Send valid hydration first
	validHydrationJSON := `{
		"events": [{
			"type": "hydration",
			"eventId": 1,
			"features": [
				{"name": "test", "enabled": true, "strategies": []}
			],
			"segments": []
		}]
	}`
	
	var validDelta api.ClientFeaturesDelta
	err := json.Unmarshal([]byte(validHydrationJSON), &validDelta)
	assert.NoError(t, err)
	
	err = processor.processDelta(&validDelta)
	assert.NoError(t, err)
	
	// Verify feature exists
	if _, exists := storage.Get("test"); !exists {
		t.Error("test feature should exist after valid hydration")
	}
	
	// Try to process nil delta (should handle gracefully)
	err = processor.processDelta(nil)
	if err == nil {
		t.Error("Processing nil delta should return an error")
	}
	
	// Process delta with unknown event type (should be ignored)
	unknownEventJSON := `{
		"events": [{
			"type": "unknown-event",
			"eventId": 2
		}]
	}`
	
	var unknownDelta api.ClientFeaturesDelta
	err = json.Unmarshal([]byte(unknownEventJSON), &unknownDelta)
	assert.NoError(t, err)
	
	err = processor.processDelta(&unknownDelta)
	// Should not error, just ignore unknown event
	assert.NoError(t, err)
	
	// Process valid update after error
	validUpdateJSON := `{
		"events": [{
			"type": "feature-updated",
			"eventId": 3,
			"feature": {"name": "test2", "enabled": true, "strategies": []}
		}]
	}`
	
	var validUpdateDelta api.ClientFeaturesDelta
	err = json.Unmarshal([]byte(validUpdateJSON), &validUpdateDelta)
	assert.NoError(t, err)
	
	err = processor.processDelta(&validUpdateDelta)
	assert.NoError(t, err)
	
	// Both features should exist
	if _, exists := storage.Get("test"); !exists {
		t.Error("test feature should still exist")
	}
	if _, exists := storage.Get("test2"); !exists {
		t.Error("test2 feature should exist after valid update")
	}
}

// TestStreamingDelta_ConstraintEvaluation tests constraint evaluation for segments
func TestStreamingDelta_ConstraintEvaluation(t *testing.T) {
	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")
	
	options := repositoryOptions{
		storage: storage,
	}
	
	repo := &repository{
		segments: make(map[int][]api.Constraint),
		options:  options,
	}
	
	channels := repositoryChannels{
		ready:  make(chan bool, 1),
		update: make(chan bool, 1),
		errorChannels: errorChannels{
			errors:   make(chan error, 10),
			warnings: make(chan error, 10),
		},
	}
	
	processor := newStreamingProcessor(storage, repo, channels)
	
	// Setup feature with complex segment constraints
	hydrationJSON := `{
		"events": [{
			"type": "hydration",
			"eventId": 1,
			"features": [
				{
					"name": "complex-segment-feature",
					"enabled": true,
					"strategies": [{
						"name": "default",
						"segments": [1]
					}]
				}
			],
			"segments": [
				{
					"id": 1,
					"constraints": [
						{
							"contextName": "userId",
							"operator": "IN",
							"values": ["123", "456"]
						},
						{
							"contextName": "environment",
							"operator": "NOT_IN",
							"values": ["dev"]
						}
					]
				}
			]
		}]
	}`
	
	var hydrationDelta api.ClientFeaturesDelta
	err := json.Unmarshal([]byte(hydrationJSON), &hydrationDelta)
	assert.NoError(t, err)
	
	err = processor.processDelta(&hydrationDelta)
	assert.NoError(t, err)
	
	// Test constraint evaluation logic would go here
	// This is primarily testing that the segments are properly stored
	if constraints, exists := repo.segments[1]; !exists {
		t.Error("Segment 1 should exist")
	} else if len(constraints) != 2 {
		t.Errorf("Segment 1 should have 2 constraints, got %d", len(constraints))
	} else {
		// Verify first constraint (userId IN)
		if constraints[0].ContextName != "userId" {
			t.Errorf("First constraint should be userId, got %s", constraints[0].ContextName)
		}
		if constraints[0].Operator != "IN" {
			t.Errorf("First constraint should be IN, got %s", constraints[0].Operator)
		}
		if len(constraints[0].Values) != 2 {
			t.Errorf("First constraint should have 2 values, got %d", len(constraints[0].Values))
		}
		
		// Verify second constraint (environment NOT_IN)
		if constraints[1].ContextName != "environment" {
			t.Errorf("Second constraint should be environment, got %s", constraints[1].ContextName)
		}
		if constraints[1].Operator != "NOT_IN" {
			t.Errorf("Second constraint should be NOT_IN, got %s", constraints[1].Operator)
		}
	}
}

// TestStreamingDelta_FeatureEvaluation tests feature evaluation with real context
func TestStreamingDelta_FeatureEvaluation(t *testing.T) {
	// This test validates that features can be evaluated with context after processing deltas
	storage := &DefaultStorage{}
	storage.Init("/tmp", "test-app")
	
	options := repositoryOptions{
		storage: storage,
	}
	
	repo := &repository{
		segments: make(map[int][]api.Constraint),
		options:  options,
	}
	
	channels := repositoryChannels{
		ready:  make(chan bool, 1),
		update: make(chan bool, 1),
		errorChannels: errorChannels{
			errors:   make(chan error, 10),
			warnings: make(chan error, 10),
		},
	}
	
	processor := newStreamingProcessor(storage, repo, channels)
	
	// Setup features with various strategies
	hydrationJSON := `{
		"events": [{
			"type": "hydration",
			"eventId": 1,
			"features": [
				{
					"name": "always-on",
					"enabled": true,
					"strategies": [{"name": "default"}]
				},
				{
					"name": "user-based",
					"enabled": true,
					"strategies": [{
						"name": "userWithId",
						"parameters": {
							"userIds": "user1,user2,user3"
						}
					}]
				},
				{
					"name": "disabled-feature",
					"enabled": false,
					"strategies": [{"name": "default"}]
				}
			],
			"segments": []
		}]
	}`
	
	var hydrationDelta api.ClientFeaturesDelta
	err := json.Unmarshal([]byte(hydrationJSON), &hydrationDelta)
	assert.NoError(t, err)
	
	err = processor.processDelta(&hydrationDelta)
	assert.NoError(t, err)
	
	// Create a mock client to test feature evaluation
	// Note: In a real integration test, you'd use the actual Client
	// Here we're just verifying the data is stored correctly
	
	// Verify always-on feature
	if feature, exists := storage.Get("always-on"); !exists {
		t.Error("always-on feature should exist")
	} else if f, ok := feature.(api.Feature); ok {
		assert.True(t, f.Enabled, "always-on should be enabled")
		assert.Equal(t, 1, len(f.Strategies), "always-on should have 1 strategy")
		assert.Equal(t, "default", f.Strategies[0].Name, "always-on should use default strategy")
	}
	
	// Verify user-based feature
	if feature, exists := storage.Get("user-based"); !exists {
		t.Error("user-based feature should exist")
	} else if f, ok := feature.(api.Feature); ok {
		assert.True(t, f.Enabled, "user-based should be enabled")
		assert.Equal(t, 1, len(f.Strategies), "user-based should have 1 strategy")
		assert.Equal(t, "userWithId", f.Strategies[0].Name, "user-based should use userWithId strategy")
		assert.Equal(t, "user1,user2,user3", f.Strategies[0].Parameters["userIds"], "userIds parameter should match")
	}
	
	// Verify disabled feature
	if feature, exists := storage.Get("disabled-feature"); !exists {
		t.Error("disabled-feature should exist")
	} else if f, ok := feature.(api.Feature); ok {
		assert.False(t, f.Enabled, "disabled-feature should be disabled")
	}
	
	// Now update a feature and verify the change
	updateJSON := `{
		"events": [{
			"type": "feature-updated",
			"eventId": 2,
			"feature": {
				"name": "disabled-feature",
				"enabled": true,
				"strategies": [{"name": "default"}]
			}
		}]
	}`
	
	var updateDelta api.ClientFeaturesDelta
	err = json.Unmarshal([]byte(updateJSON), &updateDelta)
	assert.NoError(t, err)
	
	err = processor.processDelta(&updateDelta)
	assert.NoError(t, err)
	
	// Verify the feature is now enabled
	if feature, exists := storage.Get("disabled-feature"); !exists {
		t.Error("disabled-feature should still exist")
	} else if f, ok := feature.(api.Feature); ok {
		assert.True(t, f.Enabled, "disabled-feature should now be enabled")
	}
}

