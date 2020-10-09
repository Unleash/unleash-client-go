package api

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

// Constraint represents a constraint on a particular context value.
type Constraint struct {
	// ContextName is the context name of the constraint.
	ContextName string   `json:"contextName"`

	// Operator is the operator of the constraint.
	Operator    Operator `json:"operator"`

	// Values is the values of the constraint.
	Values      []string `json:"values"`
}
