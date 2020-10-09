package constraints

import (
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
)

func checkConstraint(ctx *context.Context, constraint api.Constraint) bool {
	field := constraint.ContextName
	value := ctx.Field(field)
	isIn := false
	for _, s := range constraint.Values {
		if s == value {
			isIn = true
			break
		}
	}
	return (constraint.Operator == api.OperatorIn) == isIn
}

// Check validates all of the constraints and returns true if all constraints are satisfied.
func Check(ctx *context.Context, constraints []api.Constraint) bool {
	for _, c := range constraints {
		if !checkConstraint(ctx, c) {
			return false
		}
	}
	return true
}
