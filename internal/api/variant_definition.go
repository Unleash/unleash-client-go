package api

import "github.com/Unleash/unleash-client-go/v3/payload"

type VariantDefinition struct {
	Name    string          `json:"name"`
	Weight  int             `json:"weight"`
	Payload payload.Payload `json:"payload"`
}
