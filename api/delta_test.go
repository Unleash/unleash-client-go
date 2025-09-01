package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseDeltaEvent(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantType  string
		wantErr   bool
		wantCount int
	}{
		{
			name: "feature-updated event",
			json: `{
				"events": [{
					"type": "feature-updated",
					"eventId": 1,
					"feature": {
						"name": "test-feature",
						"enabled": true,
						"strategies": [],
						"variants": []
					}
				}]
			}`,
			wantType:  "feature-updated",
			wantCount: 1,
		},
		{
			name: "feature-removed event",
			json: `{
				"events": [{
					"type": "feature-removed",
					"eventId": 2,
					"featureName": "old-feature",
					"project": "default"
				}]
			}`,
			wantType:  "feature-removed",
			wantCount: 1,
		},
		{
			name: "segment-updated event",
			json: `{
				"events": [{
					"type": "segment-updated",
					"eventId": 3,
					"segment": {
						"id": 1,
						"constraints": [
							{
								"contextName": "userId",
								"operator": "IN",
								"values": ["123", "456"]
							}
						]
					}
				}]
			}`,
			wantType:  "segment-updated",
			wantCount: 1,
		},
		{
			name: "segment-removed event",
			json: `{
				"events": [{
					"type": "segment-removed",
					"eventId": 4,
					"segmentId": 1
				}]
			}`,
			wantType:  "segment-removed",
			wantCount: 1,
		},
		{
			name: "hydration event",
			json: `{
				"events": [{
					"type": "hydration",
					"eventId": 5,
					"features": [
						{
							"name": "feature-1",
							"enabled": true,
							"strategies": []
						},
						{
							"name": "feature-2",
							"enabled": false,
							"strategies": []
						}
					],
					"segments": [
						{
							"id": 1,
							"constraints": []
						}
					]
				}]
			}`,
			wantType:  "hydration",
			wantCount: 1,
		},
		{
			name: "multiple events",
			json: `{
				"events": [
					{
						"type": "feature-updated",
						"eventId": 1,
						"feature": {
							"name": "feature-1",
							"enabled": true
						}
					},
					{
						"type": "feature-removed",
						"eventId": 2,
						"featureName": "feature-2",
						"project": "default"
					}
				]
			}`,
			wantType:  "feature-updated", // First event type
			wantCount: 2,
		},
		{
			name:      "malformed JSON",
			json:      `{invalid json}`,
			wantErr:   true,
			wantCount: 0,
		},
		{
			name:      "missing events array",
			json:      `{}`,
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var delta ClientFeaturesDelta
			err := json.Unmarshal([]byte(tt.json), &delta)

			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if len(delta.Events) != tt.wantCount {
				t.Errorf("Got %d events, want %d", len(delta.Events), tt.wantCount)
			}

			if tt.wantCount > 0 && delta.Events[0].GetType() != tt.wantType {
				t.Errorf("First event type = %v, want %v", delta.Events[0].GetType(), tt.wantType)
			}
		})
	}
}

func TestDeltaEventTypes(t *testing.T) {
	t.Run("FeatureUpdated", func(t *testing.T) {
		jsonStr := `{
			"type": "feature-updated",
			"eventId": 1,
			"feature": {
				"name": "test-feature",
				"enabled": true,
				"type": "release",
				"description": "Test feature",
				"strategies": [
					{
						"name": "default",
						"parameters": {}
					}
				],
				"variants": [
					{
						"name": "variant1",
						"weight": 100
					}
				],
				"dependencies": [
					{
						"feature": "parent-feature",
						"enabled": true
					}
				]
			}
		}`

		var event FeatureUpdatedEvent
		err := json.Unmarshal([]byte(jsonStr), &event)
		if err != nil {
			t.Fatalf("Failed to unmarshal FeatureUpdatedEvent: %v", err)
		}

		if event.Type != "feature-updated" {
			t.Errorf("Type = %v, want feature-updated", event.Type)
		}
		if event.EventId != 1 {
			t.Errorf("EventId = %v, want 1", event.EventId)
		}
		if event.Feature.Name != "test-feature" {
			t.Errorf("Feature.Name = %v, want test-feature", event.Feature.Name)
		}
		if !event.Feature.Enabled {
			t.Error("Feature.Enabled = false, want true")
		}
		if len(event.Feature.Strategies) != 1 {
			t.Errorf("Feature.Strategies length = %v, want 1", len(event.Feature.Strategies))
		}
		if len(event.Feature.Variants) != 1 {
			t.Errorf("Feature.Variants length = %v, want 1", len(event.Feature.Variants))
		}
		if event.Feature.Dependencies == nil || len(*event.Feature.Dependencies) != 1 {
			t.Error("Feature.Dependencies not parsed correctly")
		}
	})

	t.Run("FeatureRemoved", func(t *testing.T) {
		jsonStr := `{
			"type": "feature-removed",
			"eventId": 2,
			"featureName": "deprecated-feature",
			"project": "custom-project"
		}`

		var event FeatureRemovedEvent
		err := json.Unmarshal([]byte(jsonStr), &event)
		if err != nil {
			t.Fatalf("Failed to unmarshal FeatureRemovedEvent: %v", err)
		}

		if event.Type != "feature-removed" {
			t.Errorf("Type = %v, want feature-removed", event.Type)
		}
		if event.EventId != 2 {
			t.Errorf("EventId = %v, want 2", event.EventId)
		}
		if event.FeatureName != "deprecated-feature" {
			t.Errorf("FeatureName = %v, want deprecated-feature", event.FeatureName)
		}
		if event.Project != "custom-project" {
			t.Errorf("Project = %v, want custom-project", event.Project)
		}
	})

	t.Run("SegmentUpdated", func(t *testing.T) {
		jsonStr := `{
			"type": "segment-updated",
			"eventId": 3,
			"segment": {
				"id": 42,
				"constraints": [
					{
						"contextName": "environment",
						"operator": "IN",
						"values": ["production", "staging"]
					}
				]
			}
		}`

		var event SegmentUpdatedEvent
		err := json.Unmarshal([]byte(jsonStr), &event)
		if err != nil {
			t.Fatalf("Failed to unmarshal SegmentUpdatedEvent: %v", err)
		}

		if event.Type != "segment-updated" {
			t.Errorf("Type = %v, want segment-updated", event.Type)
		}
		if event.EventId != 3 {
			t.Errorf("EventId = %v, want 3", event.EventId)
		}
		if event.Segment.Id != 42 {
			t.Errorf("Segment.Id = %v, want 42", event.Segment.Id)
		}
		if len(event.Segment.Constraints) != 1 {
			t.Errorf("Segment.Constraints length = %v, want 1", len(event.Segment.Constraints))
		}
	})

	t.Run("SegmentRemoved", func(t *testing.T) {
		jsonStr := `{
			"type": "segment-removed",
			"eventId": 4,
			"segmentId": 99
		}`

		var event SegmentRemovedEvent
		err := json.Unmarshal([]byte(jsonStr), &event)
		if err != nil {
			t.Fatalf("Failed to unmarshal SegmentRemovedEvent: %v", err)
		}

		if event.Type != "segment-removed" {
			t.Errorf("Type = %v, want segment-removed", event.Type)
		}
		if event.EventId != 4 {
			t.Errorf("EventId = %v, want 4", event.EventId)
		}
		if event.SegmentId != 99 {
			t.Errorf("SegmentId = %v, want 99", event.SegmentId)
		}
	})

	t.Run("Hydration", func(t *testing.T) {
		jsonStr := `{
			"type": "hydration",
			"eventId": 5,
			"features": [
				{
					"name": "feature-1",
					"enabled": true
				},
				{
					"name": "feature-2",
					"enabled": false
				}
			],
			"segments": [
				{
					"id": 1,
					"constraints": []
				},
				{
					"id": 2,
					"constraints": []
				}
			]
		}`

		var event HydrationEvent
		err := json.Unmarshal([]byte(jsonStr), &event)
		if err != nil {
			t.Fatalf("Failed to unmarshal HydrationEvent: %v", err)
		}

		if event.Type != "hydration" {
			t.Errorf("Type = %v, want hydration", event.Type)
		}
		if event.EventId != 5 {
			t.Errorf("EventId = %v, want 5", event.EventId)
		}
		if len(event.Features) != 2 {
			t.Errorf("Features length = %v, want 2", len(event.Features))
		}
		if len(event.Segments) != 2 {
			t.Errorf("Segments length = %v, want 2", len(event.Segments))
		}
	})
}

func TestEventOrdering(t *testing.T) {
	jsonStr := `{
		"events": [
			{"type": "feature-updated", "eventId": 3, "feature": {"name": "f3", "enabled": true}},
			{"type": "feature-updated", "eventId": 1, "feature": {"name": "f1", "enabled": true}},
			{"type": "feature-updated", "eventId": 2, "feature": {"name": "f2", "enabled": true}}
		]
	}`

	var delta ClientFeaturesDelta
	err := json.Unmarshal([]byte(jsonStr), &delta)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Events should maintain their order as received
	expectedIds := []int{3, 1, 2}
	for i, event := range delta.Events {
		if event.GetEventId() != expectedIds[i] {
			t.Errorf("Event %d has ID %d, want %d", i, event.GetEventId(), expectedIds[i])
		}
	}
}

func TestDeltaEventInterface(t *testing.T) {
	tests := []struct {
		name     string
		event    DeltaEvent
		wantType string
		wantId   int
	}{
		{
			name:     "FeatureUpdatedEvent",
			event:    &FeatureUpdatedEvent{Type: "feature-updated", EventId: 1},
			wantType: "feature-updated",
			wantId:   1,
		},
		{
			name:     "FeatureRemovedEvent",
			event:    &FeatureRemovedEvent{Type: "feature-removed", EventId: 2},
			wantType: "feature-removed",
			wantId:   2,
		},
		{
			name:     "SegmentUpdatedEvent",
			event:    &SegmentUpdatedEvent{Type: "segment-updated", EventId: 3},
			wantType: "segment-updated",
			wantId:   3,
		},
		{
			name:     "SegmentRemovedEvent",
			event:    &SegmentRemovedEvent{Type: "segment-removed", EventId: 4},
			wantType: "segment-removed",
			wantId:   4,
		},
		{
			name:     "HydrationEvent",
			event:    &HydrationEvent{Type: "hydration", EventId: 5},
			wantType: "hydration",
			wantId:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.GetType(); got != tt.wantType {
				t.Errorf("GetType() = %v, want %v", got, tt.wantType)
			}
			if got := tt.event.GetEventId(); got != tt.wantId {
				t.Errorf("GetEventId() = %v, want %v", got, tt.wantId)
			}
		})
	}
}

func TestUnmarshalDeltaEvents(t *testing.T) {
	jsonStr := `{
		"events": [
			{"type": "feature-updated", "eventId": 1, "feature": {"name": "f1", "enabled": true}},
			{"type": "feature-removed", "eventId": 2, "featureName": "f2", "project": "default"},
			{"type": "segment-updated", "eventId": 3, "segment": {"id": 1, "constraints": []}},
			{"type": "segment-removed", "eventId": 4, "segmentId": 2},
			{"type": "hydration", "eventId": 5, "features": [], "segments": []}
		]
	}`

	var delta ClientFeaturesDelta
	err := json.Unmarshal([]byte(jsonStr), &delta)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(delta.Events) != 5 {
		t.Fatalf("Expected 5 events, got %d", len(delta.Events))
	}

	// Verify each event type was properly unmarshaled
	expectedTypes := []string{
		"feature-updated",
		"feature-removed",
		"segment-updated",
		"segment-removed",
		"hydration",
	}

	for i, event := range delta.Events {
		if event.GetType() != expectedTypes[i] {
			t.Errorf("Event %d: type = %v, want %v", i, event.GetType(), expectedTypes[i])
		}
		if event.GetEventId() != i+1 {
			t.Errorf("Event %d: eventId = %v, want %v", i, event.GetEventId(), i+1)
		}
	}

	// Type assert specific events to verify fields
	if fu, ok := delta.Events[0].(*FeatureUpdatedEvent); ok {
		if fu.Feature.Name != "f1" {
			t.Errorf("FeatureUpdated feature name = %v, want f1", fu.Feature.Name)
		}
	} else {
		t.Error("First event should be FeatureUpdatedEvent")
	}

	if fr, ok := delta.Events[1].(*FeatureRemovedEvent); ok {
		if fr.FeatureName != "f2" {
			t.Errorf("FeatureRemoved feature name = %v, want f2", fr.FeatureName)
		}
	} else {
		t.Error("Second event should be FeatureRemovedEvent")
	}
}

func TestEmptyDelta(t *testing.T) {
	jsonStr := `{"events": []}`
	
	var delta ClientFeaturesDelta
	err := json.Unmarshal([]byte(jsonStr), &delta)
	if err != nil {
		t.Fatalf("Failed to unmarshal empty delta: %v", err)
	}

	if len(delta.Events) != 0 {
		t.Errorf("Expected 0 events, got %d", len(delta.Events))
	}
}

func TestDeltaWithUnknownEventType(t *testing.T) {
	jsonStr := `{
		"events": [
			{"type": "unknown-event", "eventId": 1, "someField": "value"},
			{"type": "feature-updated", "eventId": 2, "feature": {"name": "f1", "enabled": true}}
		]
	}`

	var delta ClientFeaturesDelta
	err := json.Unmarshal([]byte(jsonStr), &delta)
	
	// Should handle unknown event types gracefully
	if err != nil {
		t.Logf("Unmarshal with unknown event type: %v", err)
	}
	
	// Should still process known events
	knownEvents := 0
	for _, event := range delta.Events {
		if event != nil && event.GetType() == "feature-updated" {
			knownEvents++
		}
	}
	
	if knownEvents != 1 {
		t.Errorf("Expected 1 known event, got %d", knownEvents)
	}
}

func TestDeltaEventComparison(t *testing.T) {
	event1 := &FeatureUpdatedEvent{
		Type:    "feature-updated",
		EventId: 1,
		Feature: Feature{Name: "test"},
	}
	
	event2 := &FeatureUpdatedEvent{
		Type:    "feature-updated",
		EventId: 1,
		Feature: Feature{Name: "test"},
	}
	
	event3 := &FeatureUpdatedEvent{
		Type:    "feature-updated",
		EventId: 2,
		Feature: Feature{Name: "test"},
	}
	
	// Same event ID and content
	if !reflect.DeepEqual(event1, event2) {
		t.Error("Expected events with same ID and content to be equal")
	}
	
	// Different event ID
	if reflect.DeepEqual(event1, event3) {
		t.Error("Expected events with different IDs to be different")
	}
}