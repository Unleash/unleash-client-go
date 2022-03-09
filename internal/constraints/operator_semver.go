package constraints

import (
	"github.com/Masterminds/semver/v3"
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
)

func operatorSemverEq(ctx *context.Context, constraint api.Constraint) (bool, error) {
	return operatorSemver(ctx, constraint, func(context *semver.Version, constraint *semver.Version) bool {
		return context.Equal(constraint)
	})
}

func operatorSemverLt(ctx *context.Context, constraint api.Constraint) (bool, error) {
	return operatorSemver(ctx, constraint, func(context *semver.Version, constraint *semver.Version) bool {
		return context.LessThan(constraint)
	})
}

func operatorSemverGt(ctx *context.Context, constraint api.Constraint) (bool, error) {
	return operatorSemver(ctx, constraint, func(context *semver.Version, constraint *semver.Version) bool {
		return context.GreaterThan(constraint)
	})
}

func operatorSemver(
	ctx *context.Context,
	constraint api.Constraint,
	check func(context *semver.Version, constraint *semver.Version) bool,
) (bool, error) {
	contextValue := ctx.Field(constraint.ContextName)
	contextParsed, contextErr := semver.NewVersion(contextValue)
	constraintParsed, constraintErr := semver.NewVersion(constraint.Value)

	if contextErr != nil {
		return false, contextErr
	}

	if constraintErr != nil {
		return false, constraintErr
	}

	return check(contextParsed, constraintParsed), nil
}
