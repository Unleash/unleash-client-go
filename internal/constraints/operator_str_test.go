package constraints

import (
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperatorStr(t *testing.T) {
	testCases := []checkTestCase{
		{
			ctx:         &context.Context{UserId: "123"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_CONTAINS", Values: []string{"abc"}}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "abc"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_CONTAINS", Values: []string{"abc"}}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "abc123"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_CONTAINS", Values: []string{"abc"}}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "ABC"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_CONTAINS", Values: []string{"abc"}}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "ABC"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_CONTAINS", Values: []string{"abc"}, CaseInsensitive: true}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "123abc"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_STARTS_WITH", Values: []string{"abc"}}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "abc123"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_STARTS_WITH", Values: []string{"abc"}}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: ""},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_ENDS_WITH", Values: []string{"abc"}}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "abc"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_ENDS_WITH", Values: []string{""}}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "abc"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_ENDS_WITH", Values: []string{"abc"}}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "abc123"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_ENDS_WITH", Values: []string{"abc"}}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "123abc"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "STR_ENDS_WITH", Values: []string{"abc"}}},
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
