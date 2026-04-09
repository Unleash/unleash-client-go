package api

import (
	"encoding/json"
	"fmt"
)

type ResponseKind int

const (
	FullResponse = iota
	DeltaResponse
)

type ApiResponse struct {
	Full  *FeatureResponse
	Delta *ClientFeaturesDelta
	Kind  ResponseKind
}

func (r *ApiResponse) IsFullResponse() bool {
	return r.Kind == FullResponse
}

func (r *ApiResponse) IsDeltaResponse() bool {
	return r.Kind == DeltaResponse
}

func (r *ApiResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if events, ok := raw["events"]; ok && events != nil {
		var delta ClientFeaturesDelta
		if err := json.Unmarshal(data, &delta); err != nil {
			return err
		}
		r.Delta = &delta
		r.Full = nil
		r.Kind = DeltaResponse
		return nil
	}

	if features, ok := raw["features"]; ok && features != nil {
		var full FeatureResponse
		if err := json.Unmarshal(data, &full); err != nil {
			return err
		}
		r.Full = &full
		r.Delta = nil
		r.Kind = FullResponse
		return nil
	}

	return fmt.Errorf("unknown response format: %s", string(data))
}
