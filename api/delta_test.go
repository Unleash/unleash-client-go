package api

import (
	"encoding/json"
	"testing"
)

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
		id, ok := eventID(event)
		if !ok {
			t.Fatalf("Event %d has unsupported type %T", i, event)
		}
		if id != expectedIds[i] {
			t.Errorf("Event %d has ID %d, want %d", i, id, expectedIds[i])
		}
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
		if event == nil {
			continue
		}
		if _, ok := event.(*FeatureUpdatedEvent); ok {
			knownEvents++
		}
	}

	if knownEvents != 1 {
		t.Errorf("Expected 1 known event, got %d", knownEvents)
	}
}

func eventID(event DeltaEvent) (int, bool) {
	switch e := event.(type) {
	case *FeatureUpdatedEvent:
		return e.EventId, true
	case *FeatureRemovedEvent:
		return e.EventId, true
	case *SegmentUpdatedEvent:
		return e.EventId, true
	case *SegmentRemovedEvent:
		return e.EventId, true
	case *HydrationEvent:
		return e.EventId, true
	default:
		return 0, false
	}
}

func TestApplyFeatureUpdatedEvent(t *testing.T) {
	state := &DeltaState{
		Features: map[string]*Feature{},
		Segments: map[int][]Constraint{},
	}
	event := &FeatureUpdatedEvent{
		Feature: Feature{Name: "flag-a", Enabled: true},
	}

	event.Apply(state)

	if got, ok := state.Features["flag-a"]; !ok || got == nil || !got.Enabled {
		t.Fatalf("expected feature to be applied and enabled")
	}
}

func TestApplyFeatureRemovedEvent(t *testing.T) {
	state := &DeltaState{
		Features: map[string]*Feature{
			"flag-a": {Name: "flag-a", Enabled: true},
		},
		Segments: map[int][]Constraint{},
	}
	event := &FeatureRemovedEvent{FeatureName: "flag-a"}

	event.Apply(state)

	if _, ok := state.Features["flag-a"]; ok {
		t.Fatalf("expected feature to be removed")
	}
}

func TestApplySegmentUpdatedEvent(t *testing.T) {
	state := &DeltaState{
		Features: map[string]*Feature{},
		Segments: map[int][]Constraint{},
	}
	event := &SegmentUpdatedEvent{
		Segment: Segment{Id: 1, Constraints: []Constraint{{ContextName: "userId", Operator: "IN", Values: []string{"123"}}}},
	}

	event.Apply(state)

	if got, ok := state.Segments[1]; !ok || len(got) != 1 {
		t.Fatalf("expected segment constraints to be applied")
	}
}

func TestApplySegmentRemovedEvent(t *testing.T) {
	state := &DeltaState{
		Features: map[string]*Feature{},
		Segments: map[int][]Constraint{2: {{ContextName: "userId", Operator: "IN", Values: []string{"123"}}}},
	}
	event := &SegmentRemovedEvent{SegmentId: 2}

	event.Apply(state)

	if _, ok := state.Segments[2]; ok {
		t.Fatalf("expected segment to be removed")
	}
}

func TestApplyHydrationEvent(t *testing.T) {
	state := &DeltaState{
		Features: map[string]*Feature{
			"old": {Name: "old", Enabled: true},
		},
		Segments: map[int][]Constraint{99: {{ContextName: "userId", Operator: "IN", Values: []string{"old"}}}},
	}
	event := &HydrationEvent{
		Features: []Feature{
			{Name: "new", Enabled: false},
		},
		Segments: []Segment{
			{Id: 3, Constraints: []Constraint{{ContextName: "sessionId", Operator: "IN", Values: []string{"abc"}}}},
		},
	}

	event.Apply(state)

	if len(state.Features) != 1 {
		t.Fatalf("expected features to be replaced on hydration")
	}
	if _, ok := state.Features["new"]; !ok {
		t.Fatalf("expected hydrated feature to be present")
	}
	if len(state.Segments) != 1 {
		t.Fatalf("expected segments to be replaced on hydration")
	}
	if _, ok := state.Segments[3]; !ok {
		t.Fatalf("expected hydrated segment to be present")
	}
}
