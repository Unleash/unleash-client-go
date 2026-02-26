package constraints

import (
	"regexp"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/Unleash/unleash-go-sdk/v6/context"
)

func operatorRegex(ctx *context.Context, constraint api.Constraint) bool {
	contextValue := ctx.Field(constraint.ContextName)

	pattern := constraint.Value
	if constraint.CaseInsensitive {
		pattern = "(?i:" + pattern + ")"
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}

	return re.MatchString(contextValue)
}
