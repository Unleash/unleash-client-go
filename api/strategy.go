package api

type StrategyResponse struct {
	Response
	Strategies []StrategyDescription `json:"strategies"`
}

type Strategy struct {
	// Id is the name of the strategy.
	Id int `json:"id"`

	// Name is the name of the strategy.
	Name string `json:"name"`

	// Constraints is the constraints of the strategy.
	Constraints []Constraint `json:"constraints"`

	// Parameters is the parameters of the strategy.
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
