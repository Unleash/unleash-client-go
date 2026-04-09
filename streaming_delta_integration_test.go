package unleash

import (
	"encoding/json"
	"testing"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/stretchr/testify/assert"
)

// TestStreamingDeltaIntegration tests the full delta processing flow
func TestStreamingDeltaIntegration(t *testing.T) {
	processor := newFeatureCache(nil)

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

		_, err = processor.updateFromDelta(&delta)
		snapshot := processor.snapshot()
		if err != nil {
			t.Fatalf("Failed to process hydration: %v", err)
		}
		feature, exists := snapshot.Features["feature-1"]
		if !exists {
			t.Error("feature-1 should exist after hydration")
		} else if !feature.Enabled {
			t.Error("feature-1 should be enabled")
		}

		feature, exists = snapshot.Features["feature-2"]
		if !exists {
			t.Error("feature-2 should exist after hydration")
		} else if feature.Enabled {
			t.Error("feature-2 should be disabled")
		}

		segments := snapshot.Segments
		if len(segments) != 2 {
			t.Errorf("Expected 2 segments, got %d", len(segments))
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

		_, err = processor.updateFromDelta(&delta)
		if err != nil {
			t.Fatalf("Failed to process update: %v", err)
		}
		snapshot := processor.snapshot()

		feature, exists := snapshot.Features["feature-1"]
		if !exists {
			t.Error("feature-1 should still exist")
		} else if feature.Enabled {
			t.Error("feature-1 should be disabled after update")
		}

		if _, exists := snapshot.Features["feature-2"]; exists {
			t.Error("feature-2 should not exist after removal")
		}

		feature, exists = snapshot.Features["feature-3"]
		if !exists {
			t.Error("feature-3 should exist after update")
		} else if !feature.Enabled {
			t.Error("feature-3 should be enabled")
		}

		segments := snapshot.Segments
		if len(segments) != 2 {
			t.Errorf("Expected 2 segments after updates, got %d", len(segments))
		}

		if _, exists := segments[1]; exists {
			t.Error("Segment 1 should not exist after removal")
		}

		if _, exists := segments[3]; !exists {
			t.Error("Segment 3 should exist after update")
		}
	})
}

// TestStreamingDelta_MultipleDeltaEvents tests processing multiple delta events in sequence
func TestStreamingDelta_MultipleDeltaEvents(t *testing.T) {
	processor := newFeatureCache(nil)

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

	_, err = processor.updateFromDelta(&hydrationDelta)
	assert.NoError(t, err)
	snapshot := processor.snapshot()

	// Verify initial state
	feature, exists := snapshot.Features["feature-a"]
	if !exists {
		t.Error("feature-a should exist after hydration")
	} else if !feature.Enabled {
		t.Error("feature-a should be enabled")
	}

	feature, exists = snapshot.Features["feature-b"]
	if !exists {
		t.Error("feature-b should exist after hydration")
	} else if !feature.Enabled {
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

	_, err = processor.updateFromDelta(&updatesDelta)
	assert.NoError(t, err)
	snapshot = processor.snapshot()

	// Verify final state
	feature, exists = snapshot.Features["feature-a"]
	if !exists {
		t.Error("feature-a should still exist")
	} else if feature.Enabled {
		t.Error("feature-a should be disabled after update")
	}

	_, exists = snapshot.Features["feature-b"]
	if exists {
		t.Error("feature-b should not exist after removal")
	}

	feature, exists = snapshot.Features["feature-c"]
	if !exists {
		t.Error("feature-c should exist after update")
	} else if !feature.Enabled {
		t.Error("feature-c should be enabled")
	}
}

// TestStreamingDelta_SegmentUpdates tests segment updates via delta events
func TestStreamingDelta_SegmentUpdates(t *testing.T) {
	processor := newFeatureCache(nil)

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

	_, err = processor.updateFromDelta(&hydrationDelta)
	assert.NoError(t, err)

	snapshot := processor.snapshot()

	// Verify initial segment state
	if constraints, exists := snapshot.Segments[1]; !exists {
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

	_, err = processor.updateFromDelta(&segmentUpdateDelta)
	assert.NoError(t, err)

	snapshot = processor.snapshot()

	// Verify updated segment
	if constraints, exists := snapshot.Segments[1]; !exists {
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

	_, err = processor.updateFromDelta(&multiSegmentDelta)
	assert.NoError(t, err)

	snapshot = processor.snapshot()

	// Verify both segments exist
	if _, exists := snapshot.Segments[1]; !exists {
		t.Error("Segment 1 should still exist")
	}
	if _, exists := snapshot.Segments[2]; !exists {
		t.Error("Segment 2 should exist")
	}

	// Verify feature has been updated with both segments
	feature, exists := snapshot.Features["segmented-feature"]
	if !exists {
		t.Error("segmented-feature should exist")
	} else if feature.Enabled {
		if len(feature.Strategies) != 1 {
			t.Errorf("Feature should have 1 strategy, got %d", len(feature.Strategies))
		} else if len(feature.Strategies[0].Segments) != 2 {
			t.Errorf("Strategy should have 2 segments, got %d", len(feature.Strategies[0].Segments))
		}
	}
}

// TestStreamingDelta_ErrorHandling tests error handling in delta processing
func TestStreamingDelta_ErrorHandling(t *testing.T) {
	processor := newFeatureCache(nil)

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

	_, err = processor.updateFromDelta(&validDelta)
	assert.NoError(t, err)

	snapshot := processor.snapshot()

	// Verify feature exists
	if _, exists := snapshot.Features["test"]; !exists {
		t.Error("test feature should exist after valid hydration")
	}

	// Try to process nil delta (should handle gracefully)
	_, err = processor.updateFromDelta(nil)
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

	_, err = processor.updateFromDelta(&unknownDelta)
	// Should not error, just ignore unknown event
	assert.NoError(t, err)

	snapshot = processor.snapshot()

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

	_, err = processor.updateFromDelta(&validUpdateDelta)
	assert.NoError(t, err)

	snapshot = processor.snapshot()

	// Both features should exist
	if _, exists := snapshot.Features["test"]; !exists {
		t.Error("test feature should still exist")
	}
	if _, exists := snapshot.Features["test2"]; !exists {
		t.Error("test2 feature should exist after valid update")
	}
}

// TestStreamingDelta_ConstraintEvaluation tests constraint evaluation for segments
func TestStreamingDelta_ConstraintEvaluation(t *testing.T) {
	processor := newFeatureCache(nil)

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

	_, err = processor.updateFromDelta(&hydrationDelta)
	assert.NoError(t, err)

	snapshot := processor.snapshot()

	// Test constraint evaluation logic would go here
	// This is primarily testing that the segments are properly stored
	if constraints, exists := snapshot.Segments[1]; !exists {
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
