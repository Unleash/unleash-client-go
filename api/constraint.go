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

	// OperatorStrContains indicates that the context value
	// must contain the specified substring.
	OperatorStrContains Operator = "STR_CONTAINS"

	// OperatorStrStartsWith indicates that the context value
	// must have the specified prefix.
	OperatorStrStartsWith Operator = "STR_STARTS_WITH"

	// OperatorStrEndsWith indicates that the context value
	// must have the specified suffix.
	OperatorStrEndsWith Operator = "STR_ENDS_WITH"

	// OperatorNumEq indicates that the context value
	// must be equal to the specified number.
	OperatorNumEq Operator = "NUM_EQ"

	// OperatorNumLt indicates that the context value
	// must be less than the specified number.
	OperatorNumLt Operator = "NUM_LT"

	// OperatorNumLte indicates that the context value
	// must be less than or equal to the specified number.
	OperatorNumLte Operator = "NUM_LTE"

	// OperatorNumGt indicates that the context value
	// must be greater than the specified number.
	OperatorNumGt Operator = "NUM_GT"

	// OperatorNumGte indicates that the context value
	// must be greater than or equal to the specified number.
	OperatorNumGte Operator = "NUM_GTE"

	// OperatorDateBefore indicates that the context value
	// must be before the specified date.
	OperatorDateBefore Operator = "DATE_BEFORE"

	// OperatorDateAfter indicates that the context value
	// must be after the specified date.
	OperatorDateAfter Operator = "DATE_AFTER"

	// OperatorSemverEq indicates that the context value
	// must be equal to the specified SemVer version.
	OperatorSemverEq Operator = "SEMVER_EQ"

	// OperatorSemverLt indicates that the context value
	// must be less than the specified SemVer version.
	OperatorSemverLt Operator = "SEMVER_LT"

	// OperatorSemverGt indicates that the context value
	// must be greater than the specified SemVer version.
	OperatorSemverGt Operator = "SEMVER_GT"
)

// Constraint represents a constraint on a particular context value.
type Constraint struct {
	// ContextName is the context name of the constraint.
	ContextName string `json:"contextName"`

	// Operator is the operator of the constraint.
	Operator Operator `json:"operator"`

	// Values is the list of target values for multi-valued constraints.
	Values []string `json:"values"`

	// Value is the target value single-value constraints.
	Value string `json:"value"`

	// CaseInsensitive makes the string operators case-insensitive.
	CaseInsensitive bool `json:"caseInsensitive"`

	// Inverted flips the constraint check result.
	Inverted bool `json:"inverted"`
}
