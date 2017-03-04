package api

import "time"

type ToggleCount struct {
	Yes int32 `json:"yes"`
	No  int32 `json:"no"`
}

type Bucket struct {
	Start   time.Time              `json:"start"`
	Stop    time.Time              `json:"stop"`
	Toggles map[string]ToggleCount `json:"toggles"`
}

func (b Bucket) IsEmpty() bool {
	return len(b.Toggles) == 0
}
