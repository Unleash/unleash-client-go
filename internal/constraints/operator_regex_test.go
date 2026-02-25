package constraints

import (
	"testing"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/Unleash/unleash-go-sdk/v6/context"
	"github.com/stretchr/testify/assert"
)

func TestOperatorRegex(t *testing.T) {
	testCases := []checkTestCase{
		{
			ctx:         &context.Context{Properties: map[string]string{"email": "example@getunleash.ai"}},
			constraints: []api.Constraint{{ContextName: "email", Operator: "REGEX", Value: ".*@getunleash\\.ai$"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"email": "example@getunleash.ai"}},
			constraints: []api.Constraint{{ContextName: "email", Operator: "REGEX", Value: "["}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"email": "example@getunleash.ai"}},
			constraints: []api.Constraint{{ContextName: "email", Operator: "REGEX", Value: "/.*@getunleash\\.ai$/i"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{}},
			constraints: []api.Constraint{{ContextName: "userId", Operator: "REGEX", Value: "\\d+"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"email": "EXAMPLE@GETUNLEASH.AI"}},
			constraints: []api.Constraint{{ContextName: "email", Operator: "REGEX", Value: ".*@getunleash\\.ai$", CaseInsensitive: true}},
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
