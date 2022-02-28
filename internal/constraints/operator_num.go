package constraints

import (
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
	"math/big"
)

func operatorNumEq(ctx *context.Context, constraint api.Constraint) (bool, error) {
	return operatorNum(ctx, constraint, func(context *big.Float, constraint *big.Float) bool {
		return context.Cmp(constraint) == 0
	})
}

func operatorNumLt(ctx *context.Context, constraint api.Constraint) (bool, error) {
	return operatorNum(ctx, constraint, func(context *big.Float, constraint *big.Float) bool {
		return context.Cmp(constraint) < 0
	})
}

func operatorNumLte(ctx *context.Context, constraint api.Constraint) (bool, error) {
	return operatorNum(ctx, constraint, func(context *big.Float, constraint *big.Float) bool {
		return context.Cmp(constraint) <= 0
	})
}

func operatorNumGt(ctx *context.Context, constraint api.Constraint) (bool, error) {
	return operatorNum(ctx, constraint, func(context *big.Float, constraint *big.Float) bool {
		return context.Cmp(constraint) > 0
	})
}

func operatorNumGte(ctx *context.Context, constraint api.Constraint) (bool, error) {
	return operatorNum(ctx, constraint, func(context *big.Float, constraint *big.Float) bool {
		return context.Cmp(constraint) >= 0
	})
}

func operatorNum(
	ctx *context.Context,
	constraint api.Constraint,
	check func(context *big.Float, constraint *big.Float) bool,
) (bool, error) {
	contextValue := ctx.Field(constraint.ContextName)
	contextParsed, _, contextErr := big.ParseFloat(contextValue, 10, 0, big.ToNearestEven)
	constraintParsed, _, constraintErr := big.ParseFloat(constraint.Value, 10, 0, big.ToNearestEven)

	if contextErr != nil {
		return false, contextErr
	}

	if constraintErr != nil {
		return false, constraintErr
	}

	return check(contextParsed, constraintParsed), nil
}
