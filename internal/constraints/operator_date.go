package constraints

import (
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
	"time"
)

func operatorDateBefore(ctx *context.Context, constraint api.Constraint) (bool, error) {
	return operatorDate(ctx, constraint, func(context time.Time, constraint time.Time) bool {
		return context.Before(constraint)
	})
}

func operatorDateAfter(ctx *context.Context, constraint api.Constraint) (bool, error) {
	return operatorDate(ctx, constraint, func(context time.Time, constraint time.Time) bool {
		return context.After(constraint)
	})
}

func operatorDate(
	ctx *context.Context,
	constraint api.Constraint,
	check func(context time.Time, constraint time.Time) bool,
) (bool, error) {
	contextParsed, contextErr := contextDateValueOrNow(ctx, constraint)
	constraintParsed, constraintErr := time.Parse(time.RFC3339, constraint.Value)

	if contextErr != nil {
		return false, contextErr
	}

	if constraintErr != nil {
		return false, constraintErr
	}

	return check(contextParsed, constraintParsed), nil
}

func contextDateValueOrNow(ctx *context.Context, constraint api.Constraint) (time.Time, error) {
	contextValue := ctx.Field(constraint.ContextName)

	if contextValue != "" {
		return time.Parse(time.RFC3339, contextValue)
	}

	return time.Now(), nil
}
