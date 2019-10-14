package api

type Operator string

const (
	OperatorIn    Operator = "IN"
	OperatorNotIn Operator = "NOT_IN"
)

type Constraint struct {
	ContextName string   `json:"contextName"`
	Operator    Operator `json:"operator"`
	Values      []string `json:"values"`
}
