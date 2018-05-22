package api

// Feature the structure for feature that can be returned by the client
type Feature struct {
	Name        string
	Description string
	Enabled     bool
	Strategies  []Strategy
}

// Strategy a feature strategy
type Strategy struct {
	Name       string
	Parameters map[string]interface{}
}
