package feature

import "time"

type ParameterMap map[string]interface{}

// Operator is a type representing a constraint operator
type Operator string

const (
	// OperatorIn indicates that the context values must be
	// contained within those specified in the constraint.
	OperatorIn Operator = "IN"

	// OperatorNotIn indicates that the context values must
	// NOT be contained within those specified in the constraint.
	OperatorNotIn Operator = "NOT_IN"
)

// Constraint represents a constraint on a feature strategy.
type Constraint struct {
	// ContextName is the context name of the constraint.
	ContextName string

	// Operator is the operator of the constraint.
	Operator Operator

	// Values is the values of the constraint.
	Values []string
}

// Strategy represents a strategy on a feature.
type Strategy struct {
	// Id is the name of the strategy.
	Id int

	// Name is the name of the strategy.
	Name string

	// Constraints is the constraints of the strategy.
	Constraints []*Constraint

	// Parameters is the parameters of the strategy.
	Parameters ParameterMap
}

// Feature represents a feature toggle.
type Feature struct {
	// Name is the name of the feature toggle.
	Name string

	// Description is a description of the feature toggle.
	Description string

	// Enabled indicates whether the feature was enabled or not.
	Enabled bool

	// Strategies is a list of names of the strategies supported by the client.
	Strategies []*Strategy

	// CreatedAt is the creation time of the feature toggle.
	CreatedAt time.Time

	// Strategy is the strategy of the feature toggle.
	Strategy string

	// Parameters is the parameters of the feature toggle.
	Parameters ParameterMap
}
