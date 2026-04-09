package api

import (
	"encoding/json"
	"fmt"
)

type ClientFeaturesDelta struct {
	Events []DeltaEvent `json:"events"`
}

type DeltaState struct {
	Features map[string]*Feature
	Segments map[int][]Constraint
}

type DeltaEvent interface {
	Apply(state *DeltaState)
}

type BaseDeltaEvent struct {
	Type    string `json:"type"`
	EventId int    `json:"eventId"`
}

type FeatureUpdatedEvent struct {
	BaseDeltaEvent
	Feature Feature `json:"feature"`
}

type FeatureRemovedEvent struct {
	BaseDeltaEvent
	FeatureName string `json:"featureName"`
	Project     string `json:"project"`
}
type SegmentUpdatedEvent struct {
	BaseDeltaEvent
	Segment Segment `json:"segment"`
}

type SegmentRemovedEvent struct {
	BaseDeltaEvent
	SegmentId int `json:"segmentId"`
}

type HydrationEvent struct {
	BaseDeltaEvent
	Features []Feature `json:"features"`
	Segments []Segment `json:"segments"`
}

func (e *FeatureUpdatedEvent) Apply(state *DeltaState) {
	state.Features[e.Feature.Name] = &e.Feature
}

func (e *FeatureRemovedEvent) Apply(state *DeltaState) {
	delete(state.Features, e.FeatureName)
}

func (e *SegmentUpdatedEvent) Apply(state *DeltaState) {
	state.Segments[e.Segment.Id] = e.Segment.Constraints
}

func (e *SegmentRemovedEvent) Apply(state *DeltaState) {
	delete(state.Segments, e.SegmentId)
}

func (e *HydrationEvent) Apply(state *DeltaState) {
	state.Features = make(map[string]*Feature, len(e.Features))
	for _, f := range e.Features {
		state.Features[f.Name] = &f
	}

	state.Segments = make(map[int][]Constraint, len(e.Segments))
	for _, seg := range e.Segments {
		state.Segments[seg.Id] = seg.Constraints
	}
}

func unmarshalEvent[T DeltaEvent](raw json.RawMessage) (DeltaEvent, error) {
	var event T
	if err := json.Unmarshal(raw, &event); err != nil {
		return nil, err
	}
	return event, nil
}

func (c *ClientFeaturesDelta) UnmarshalJSON(data []byte) error {
	var raw struct {
		Events []json.RawMessage `json:"events"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.Events = make([]DeltaEvent, 0, len(raw.Events))

	for _, rawEvent := range raw.Events {
		var eventType struct {
			Type string `json:"type"`
		}

		if err := json.Unmarshal(rawEvent, &eventType); err != nil {
			return fmt.Errorf("failed to unmarshal event type: %w", err)
		}

		var (
			event DeltaEvent
			err   error
		)

		switch eventType.Type {
		case "feature-updated":
			event, err = unmarshalEvent[*FeatureUpdatedEvent](rawEvent)
		case "feature-removed":
			event, err = unmarshalEvent[*FeatureRemovedEvent](rawEvent)
		case "segment-updated":
			event, err = unmarshalEvent[*SegmentUpdatedEvent](rawEvent)
		case "segment-removed":
			event, err = unmarshalEvent[*SegmentRemovedEvent](rawEvent)
		case "hydration":
			event, err = unmarshalEvent[*HydrationEvent](rawEvent)
		default:
			// Unknown event type, this gives us a safety net for forward compatibility with new event types
			continue
		}

		if err != nil {
			return fmt.Errorf("failed to unmarshal %s event: %w", eventType.Type, err)
		}

		c.Events = append(c.Events, event)
	}

	return nil
}
