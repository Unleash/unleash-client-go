package constraints

import (
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperatorSemver(t *testing.T) {
	testCases := []checkTestCase{
		{
			ctx:         &context.Context{Properties: map[string]string{"v": "1.2.3"}},
			constraints: []api.Constraint{{ContextName: "v", Operator: "SEMVER_EQ", Value: "3.2.1"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"v": "1.2.3"}},
			constraints: []api.Constraint{{ContextName: "v", Operator: "SEMVER_EQ", Value: "1.2.3"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"v": "1.2.3"}},
			constraints: []api.Constraint{{ContextName: "v", Operator: "SEMVER_EQ", Value: "v1.2.3"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"v": "v1.2.3"}},
			constraints: []api.Constraint{{ContextName: "v", Operator: "SEMVER_EQ", Value: "v3.2.1"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"v": "v1.2.3"}},
			constraints: []api.Constraint{{ContextName: "v", Operator: "SEMVER_EQ", Value: "v1.2.3"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"v": "v1.2.3"}},
			constraints: []api.Constraint{{ContextName: "v", Operator: "SEMVER_EQ", Value: "v1.2.3"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"v": "v1.2.3"}},
			constraints: []api.Constraint{{ContextName: "v", Operator: "SEMVER_LT", Value: "v1.2.3"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"v": "v3.2.1"}},
			constraints: []api.Constraint{{ContextName: "v", Operator: "SEMVER_LT", Value: "v1.2.3"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"v": "v1.2.3"}},
			constraints: []api.Constraint{{ContextName: "v", Operator: "SEMVER_LT", Value: "v3.2.1"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"v": "v3.2.1"}},
			constraints: []api.Constraint{{ContextName: "v", Operator: "SEMVER_GT", Value: "v1.2.3"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"v": "v1.2.3"}},
			constraints: []api.Constraint{{ContextName: "v", Operator: "SEMVER_GT", Value: "v3.2.1"}},
			expected:    false,
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
