package unleash

import (
	"github.com/Unleash/unleash-client-go/v4/context"
)

type EventType string

const (
	ImpressionEventTypeIsEnabled EventType = "IsEnabled"
	ImpressionEventTypeGetVariant EventType = "GetVariant"
)

// ImpressionEvent represents an event that occurs when a feature with impression data is evaluated.
type ImpressionEvent struct {
	// FeatureName is the name of the feature toggle that was evaluated.
	FeatureName string

	// EventType is the type of event that occurred (i.e., "IsEnabled" or "GetVariant").
	EventType EventType

	// Enabled indicates whether the feature was enabled or not.
	Enabled bool

	// Variant is the name of the variant that was resolved, if any.
	Variant string

	// Context is the evaluation context used.
	Context *context.Context
}

