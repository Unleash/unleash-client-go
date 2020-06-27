package api

import (
	"time"

	"github.com/Unleash/unleash-client-go/v3/api"
)

type EventResponse struct {
	api.Response
	Events []Event `json:"events"`
}

type Event struct {
	Id        int          `json:"id"`
	Type      string       `json:"type"`
	CreatedBy string       `json:"createdBy"`
	CreatedAt time.Time    `json:"createdAt"`
	Data      EventData    `json:"data"`
	Diffs     *[]EventDiff `json:"diffs"`
}

type EventData struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Enabled     bool       `json:"enabled"`
	Strategies  []api.Strategy `json:"strategies"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type EventDiff struct {
	Kind string   `json:"kind"`
	Path []string `json:"path"`
	Lhs  bool     `json:"lhs"`
	Rhs  bool     `json:"rhs"`
}
