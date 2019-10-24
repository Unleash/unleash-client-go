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

// Constraint represents a constraint on a particular context
// value.
type Constraint struct {
	ContextName string   `json:"contextName"`
	Operator    Operator `json:"operator"`
	Values      []string `json:"values"`
}
