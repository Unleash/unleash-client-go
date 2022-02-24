package constraints

import (
	"fmt"
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/stretchr/testify/assert"
	"testing"
)

type checkTestCase struct {
	ctx         *context.Context
	constraints []api.Constraint
	expected    bool
}

func TestCheck(t *testing.T) {
	testCases := []checkTestCase{
		{
			expected: true,
		},
		{
			ctx:      &context.Context{},
			expected: true,
		},
		{
			ctx:         &context.Context{},
			constraints: []api.Constraint{},
			expected:    true,
		},
		{
			constraints: []api.Constraint{},
			expected:    true,
		},
		{
			ctx:         &context.Context{Environment: "dev"},
			constraints: []api.Constraint{},
			expected:    true,
		},
		{
			ctx:         &context.Context{},
			constraints: []api.Constraint{{ContextName: "environment", Operator: "IN", Values: []string{}}},
			expected:    false,
		},
		{
			ctx:         &context.Context{},
			constraints: []api.Constraint{{ContextName: "environment", Operator: "IN", Values: []string{}, Inverted: true}},
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

func TestCheckUnknownOperator(t *testing.T) {
	testCases := []checkTestCase{
		{
			ctx:         &context.Context{},
			constraints: []api.Constraint{{ContextName: "environment", Operator: "UNKNOWN_OPERATOR"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{},
			constraints: []api.Constraint{{ContextName: "environment", Operator: "UNKNOWN_OPERATOR", Inverted: true}},
			expected:    false,
		},
	}

	for _, tc := range testCases {
		ok, err := Check(tc.ctx, tc.constraints)
		assert.Equal(t, tc.expected, ok)
		assert.Equal(t, err.Error(), fmt.Sprintf("unknown constraint operator: %s", tc.constraints[0].Operator))
	}
}
