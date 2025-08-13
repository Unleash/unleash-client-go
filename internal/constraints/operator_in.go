package constraints

import (
	"github.com/Unleash/unleash-go-sdk/v5/api"
	"github.com/Unleash/unleash-go-sdk/v5/context"
	"slices"
)

func operatorNotIn(ctx *context.Context, constraint api.Constraint) bool {
	return !operatorIn(ctx, constraint)
}

func operatorIn(ctx *context.Context, constraint api.Constraint) bool {
	contextValue := ctx.Field(constraint.ContextName)

	return slices.Contains(constraint.Values, contextValue)
}
