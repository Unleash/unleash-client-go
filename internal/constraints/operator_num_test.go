package constraints

import (
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOperatorNum(t *testing.T) {
	testCases := []checkTestCase{
		{
			ctx:         &context.Context{UserId: "123"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_EQ", Value: "321"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "321"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_EQ", Value: "123"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "123"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_EQ", Value: "123"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "123.321"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_EQ", Value: "123.321"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "123.321"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_EQ", Value: "321.123"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "1.0000000000000001"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_EQ", Value: "1.0000000000000001"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "2.0000000000000001"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_EQ", Value: "2.0000000000000001"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "1.0000000000000002"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_EQ", Value: "1.0000000000000001"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "1.0000000000000001"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_EQ", Value: "1.0000000000000002"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "2.0000000000000001"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_EQ", Value: "2.0000000000000002"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "123.321"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_LT", Value: "123.321"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "321.123"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_LT", Value: "123.321"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "123.321"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_LT", Value: "321.123"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "123.321"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_LTE", Value: "123.321"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "321.123"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_LTE", Value: "123.321"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "123.321"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_LTE", Value: "321.123"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "123.321"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_GT", Value: "123.321"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "321.123"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_GT", Value: "123.321"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "123.321"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_GT", Value: "321.123"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{UserId: "123.321"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_GTE", Value: "123.321"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "321.123"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_GTE", Value: "123.321"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{UserId: "123.321"},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "NUM_GTE", Value: "321.123"}},
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
