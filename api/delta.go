package api

import (
	"encoding/json"
	"fmt"
)

// ClientFeaturesDelta represents a delta update from the streaming endpoint
type ClientFeaturesDelta struct {
	Events []DeltaEvent `json:"events"`
}

// DeltaEvent is the interface for all delta event types
type DeltaEvent interface {
	GetType() string
	GetEventId() int
}

// FeatureUpdatedEvent represents a feature update event
type FeatureUpdatedEvent struct {
	Type    string  `json:"type"`
	EventId int     `json:"eventId"`
	Feature Feature `json:"feature"`
}

func (e *FeatureUpdatedEvent) GetType() string   { return e.Type }
func (e *FeatureUpdatedEvent) GetEventId() int   { return e.EventId }

// FeatureRemovedEvent represents a feature removal event
type FeatureRemovedEvent struct {
	Type        string `json:"type"`
	EventId     int    `json:"eventId"`
	FeatureName string `json:"featureName"`
	Project     string `json:"project"`
}

func (e *FeatureRemovedEvent) GetType() string   { return e.Type }
func (e *FeatureRemovedEvent) GetEventId() int   { return e.EventId }

// SegmentUpdatedEvent represents a segment update event
type SegmentUpdatedEvent struct {
	Type    string  `json:"type"`
	EventId int     `json:"eventId"`
	Segment Segment `json:"segment"`
}

func (e *SegmentUpdatedEvent) GetType() string   { return e.Type }
func (e *SegmentUpdatedEvent) GetEventId() int   { return e.EventId }

// SegmentRemovedEvent represents a segment removal event
type SegmentRemovedEvent struct {
	Type      string `json:"type"`
	EventId   int    `json:"eventId"`
	SegmentId int    `json:"segmentId"`
}

func (e *SegmentRemovedEvent) GetType() string   { return e.Type }
func (e *SegmentRemovedEvent) GetEventId() int   { return e.EventId }

// HydrationEvent represents a full state replacement event
type HydrationEvent struct {
	Type     string    `json:"type"`
	EventId  int       `json:"eventId"`
	Features []Feature `json:"features"`
	Segments []Segment `json:"segments"`
}

func (e *HydrationEvent) GetType() string   { return e.Type }
func (e *HydrationEvent) GetEventId() int   { return e.EventId }

// UnmarshalJSON implements custom unmarshaling for ClientFeaturesDelta
func (c *ClientFeaturesDelta) UnmarshalJSON(data []byte) error {
	// First unmarshal to get the raw events
	var raw struct {
		Events []json.RawMessage `json:"events"`
	}
	
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	
	// Process each event
	c.Events = make([]DeltaEvent, 0, len(raw.Events))
	
	for _, rawEvent := range raw.Events {
		// Determine the event type
		var eventType struct {
			Type string `json:"type"`
		}
		
		if err := json.Unmarshal(rawEvent, &eventType); err != nil {
			continue // Skip malformed events
		}
		
		var event DeltaEvent
		
		switch eventType.Type {
		case "feature-updated":
			var e FeatureUpdatedEvent
			if err := json.Unmarshal(rawEvent, &e); err == nil {
				event = &e
			}
			
		case "feature-removed":
			var e FeatureRemovedEvent
			if err := json.Unmarshal(rawEvent, &e); err == nil {
				event = &e
			}
			
		case "segment-updated":
			var e SegmentUpdatedEvent
			if err := json.Unmarshal(rawEvent, &e); err == nil {
				event = &e
			}
			
		case "segment-removed":
			var e SegmentRemovedEvent
			if err := json.Unmarshal(rawEvent, &e); err == nil {
				event = &e
			}
			
		case "hydration":
			var e HydrationEvent
			if err := json.Unmarshal(rawEvent, &e); err == nil {
				event = &e
			}
			
		default:
			// Unknown event type - skip
			continue
		}
		
		if event != nil {
			c.Events = append(c.Events, event)
		}
	}
	
	return nil
}

// ParseDelta parses a JSON byte array into a ClientFeaturesDelta
func ParseDelta(data []byte) (*ClientFeaturesDelta, error) {
	var delta ClientFeaturesDelta
	if err := json.Unmarshal(data, &delta); err != nil {
		return nil, fmt.Errorf("failed to parse delta: %w", err)
	}
	return &delta, nil
}