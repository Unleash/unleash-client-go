package constraints

import (
	"github.com/Unleash/unleash-client-go/v4/api"
	"github.com/Unleash/unleash-client-go/v4/context"
)

func operatorNotIn(ctx *context.Context, constraint api.Constraint) bool {
	return !operatorIn(ctx, constraint)
}

func operatorIn(ctx *context.Context, constraint api.Constraint) bool {
	contextValue := ctx.Field(constraint.ContextName)

	for _, constraint := range constraint.Values {
		if contextValue == constraint {
			return true
		}
	}

	return false
}
