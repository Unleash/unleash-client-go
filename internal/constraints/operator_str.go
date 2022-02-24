package constraints

import (
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
	"strings"
)

func operatorStrContains(ctx *context.Context, constraint api.Constraint) bool {
	return operatorStr(ctx, constraint, strings.Contains)
}

func operatorStrStartsWith(ctx *context.Context, constraint api.Constraint) bool {
	return operatorStr(ctx, constraint, strings.HasPrefix)
}

func operatorStrEndsWith(ctx *context.Context, constraint api.Constraint) bool {
	return operatorStr(ctx, constraint, strings.HasSuffix)
}

func operatorStr(
	ctx *context.Context,
	constraint api.Constraint,
	check func(context string, constraint string) bool,
) bool {
	contextValue := toLowerIfCaseInsensitive(constraint, ctx.Field(constraint.ContextName))

	for _, constraintValue := range constraint.Values {
		if check(contextValue, toLowerIfCaseInsensitive(constraint, constraintValue)) {
			return true
		}
	}

	return false
}

func toLowerIfCaseInsensitive(constraint api.Constraint, value string) string {
	if constraint.CaseInsensitive {
		return strings.ToLower(value)
	} else {
		return value
	}
}
