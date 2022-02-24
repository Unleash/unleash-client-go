package constraints

import (
	"fmt"
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
)

// Check checks if all the constraints are fulfilled by the context.
func Check(ctx *context.Context, constraints []api.Constraint) (bool, error) {
	for _, c := range constraints {
		if ok, err := checkConstraintInvertible(ctx, c); !ok || err != nil {
			return false, err
		}
	}

	return true, nil
}

// checkConstraintInvertible inverts the result from checkConstraint if required.
func checkConstraintInvertible(ctx *context.Context, constraint api.Constraint) (bool, error) {
	ok, err := checkConstraint(ctx, constraint)

	if constraint.Inverted {
		return !ok, err
	}

	return ok, err
}

// checkConstraint checks if a specific constraint is fulfilled by the context.
func checkConstraint(ctx *context.Context, constraint api.Constraint) (bool, error) {
	switch constraint.Operator {
	case api.OperatorIn:
		return operatorIn(ctx, constraint), nil
	case api.OperatorNotIn:
		return operatorNotIn(ctx, constraint), nil
	case api.OperatorStrContains:
		return operatorStrContains(ctx, constraint), nil
	case api.OperatorStrStartsWith:
		return operatorStrStartsWith(ctx, constraint), nil
	case api.OperatorStrEndsWith:
		return operatorStrEndsWith(ctx, constraint), nil
	case api.OperatorNumEq:
		return operatorNumEq(ctx, constraint)
	case api.OperatorNumLt:
		return operatorNumLt(ctx, constraint)
	case api.OperatorNumLte:
		return operatorNumLte(ctx, constraint)
	case api.OperatorNumGt:
		return operatorNumGt(ctx, constraint)
	case api.OperatorNumGte:
		return operatorNumGte(ctx, constraint)
	case api.OperatorDateBefore:
		return operatorDateBefore(ctx, constraint)
	case api.OperatorDateAfter:
		return operatorDateAfter(ctx, constraint)
	case api.OperatorSemverEq:
		return operatorSemverEq(ctx, constraint)
	case api.OperatorSemverLt:
		return operatorSemverLt(ctx, constraint)
	case api.OperatorSemverGt:
		return operatorSemverGt(ctx, constraint)
	default:
		return false, fmt.Errorf("unknown constraint operator: %s", constraint.Operator)
	}
}
