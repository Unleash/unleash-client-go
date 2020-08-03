package constraints

import (
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheck(t *testing.T) {
	type testCase struct {
		ctx         *context.Context
		constraints []api.Constraint
		expected    bool
	}

	testCases := []testCase{
		{
			ctx: &context.Context{
				Environment: "prod",
				Properties: map[string]string{
					"country": "norway",
				},
			},
			constraints: []api.Constraint{
				{
					ContextName: "environment",
					Operator:    "IN",
					Values:      []string{"prod"},
				},
			},
			expected: true,
		},
		{
			ctx: &context.Context{
				Environment: "dev",
				Properties: map[string]string{
					"country": "norway",
				},
			},
			constraints: []api.Constraint{
				{
					ContextName: "environment",
					Operator:    "IN",
					Values:      []string{"prod"},
				},
			},
			expected: false,
		},
		{
			ctx: &context.Context{
				Environment: "prod",
			},
			constraints: []api.Constraint{
				{
					ContextName: "environment",
					Operator:    "NOT_IN",
					Values:      []string{"prod"},
				},
			},
			expected: false,
		},
		{
			ctx: &context.Context{
				Environment: "dev",
				AppName:     "test",
			},
			constraints: []api.Constraint{
				{
					ContextName: "environment",
					Operator:    "NOT_IN",
					Values:      []string{"prod"},
				},
				{
					ContextName: "appName",
					Operator:    "IN",
					Values:      []string{"test"},
				},
			},
			expected: true,
		},
		{
			ctx: &context.Context{
				Environment: "dev",
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, Check(tc.ctx, tc.constraints))
	}
}
