package api

type StrategyResponse struct {
	Response
	Strategies []StrategyDescription `json:"strategies"`
}

type Strategy struct {
	Id         int          `json:"id"`
	Name       string       `json:"name"`
	Parameters ParameterMap `json:"parameters"`
}

type ParameterDescription struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type StrategyDescription struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  []ParameterDescription `json:"parameters"`
}
