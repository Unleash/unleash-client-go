package constraints

import (
	"github.com/Unleash/unleash-client-go/v4/api"
	"github.com/Unleash/unleash-client-go/v4/context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperatorSemver(t *testing.T) {
	testCases := []checkTestCase{
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "1.2.3"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_EQ", Value: "3.2.1"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "3.2.1"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_EQ", Value: "1.2.3"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "1.2.3"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_EQ", Value: "1.2.3"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "1.2.3"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_LT", Value: "1.2.3"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "3.2.1"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_LT", Value: "1.2.3"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "1.2.3"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_LT", Value: "3.2.1"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "1.2.3"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_GT", Value: "1.2.3"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "1.2.3"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_GT", Value: "3.2.1"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "3.2.1"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_GT", Value: "1.2.3"}},
			expected:    true,
		},
	}

	for _, tc := range testCases {
		if ok, err := Check(tc.ctx, tc.constraints); err != nil {
			t.Fatal(err)
		} else {
			assert.Equal(t, tc.expected, ok)
		}
	}
}

func TestOperatorSemverErrors(t *testing.T) {
	testCases := []checkTestCase{
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "v1.2.3"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_EQ", Value: "1.2.3"}},
			error:       "Invalid characters in version",
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "1.2.3"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_EQ", Value: "v1.2.3"}},
			error:       "Invalid characters in version",
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "v1.2.3"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_LT", Value: "3.2.1"}},
			error:       "Invalid characters in version",
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "1.2.3"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_LT", Value: "v3.2.1"}},
			error:       "Invalid characters in version",
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "v3.2.1"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_GT", Value: "1.2.3"}},
			error:       "Invalid characters in version",
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "3.2.1"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_GT", Value: "v1.2.3"}},
			error:       "Invalid characters in version",
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "1.0"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_EQ", Value: "1.0.0"}},
			error:       "Invalid Semantic Version",
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"x": "1.0.0"}},
			constraints: []api.Constraint{{ContextName: "x", Operator: "SEMVER_EQ", Value: "1.0"}},
			error:       "Invalid Semantic Version",
		},
	}

	for _, tc := range testCases {
		ok, err := Check(tc.ctx, tc.constraints)
		assert.Equal(t, false, ok)
		assert.Error(t, err)
		assert.Equal(t, tc.error, err.Error())
	}
}
