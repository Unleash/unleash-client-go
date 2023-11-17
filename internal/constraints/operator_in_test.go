package constraints

import (
	"github.com/Unleash/unleash-client-go/v4/api"
	"github.com/Unleash/unleash-client-go/v4/context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperatorIn(t *testing.T) {
	testCases := []checkTestCase{
		{
			ctx:         &context.Context{Environment: "prod", Properties: map[string]string{"country": "norway"}},
			constraints: []api.Constraint{{ContextName: "environment", Operator: "IN", Values: []string{"prod"}}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Environment: "dev", Properties: map[string]string{"country": "norway"}},
			constraints: []api.Constraint{{ContextName: "environment", Operator: "IN", Values: []string{"prod"}}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Environment: "prod"},
			constraints: []api.Constraint{{ContextName: "environment", Operator: "NOT_IN", Values: []string{"prod"}}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Environment: "dev", AppName: "test"},
			constraints: []api.Constraint{{ContextName: "environment", Operator: "NOT_IN", Values: []string{"prod"}}, {ContextName: "appName", Operator: "IN", Values: []string{"test"}}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Environment: "dev", AppName: "test"},
			constraints: []api.Constraint{{ContextName: "environment", Operator: "NOT_IN", Values: []string{"prod"}}, {ContextName: "appName", Operator: "IN", Values: []string{"test"}}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Environment: "dev", AppName: "test"},
			constraints: []api.Constraint{{ContextName: "environment", Operator: "NOT_IN", Values: []string{"dev"}}, {ContextName: "appName", Operator: "IN", Values: []string{"test"}}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Environment: "dev", AppName: "test"},
			constraints: []api.Constraint{{ContextName: "environment", Operator: "IN", Values: []string{"dev"}}, {ContextName: "appName", Operator: "IN", Values: []string{"test"}}},
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
