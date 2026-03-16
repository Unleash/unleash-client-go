package constraints

import (
	"testing"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/Unleash/unleash-go-sdk/v6/context"
	"github.com/stretchr/testify/assert"
)

func TestOperatorInCidr(t *testing.T) {
	testCases := []struct {
		name       string
		ctx        *context.Context
		constraint api.Constraint
		expected   bool
	}{
		{
			name: "exact match",
			ctx:  &context.Context{RemoteAddress: "127.0.0.1"},
			constraint: api.Constraint{
				ContextName: "remoteAddress",
				Operator:    api.OperatorInCidr,
				Values:      []string{"127.0.0.1"},
			},
			expected: true,
		},
		{
			name: "cidr range match",
			ctx:  &context.Context{RemoteAddress: "160.33.0.33"},
			constraint: api.Constraint{
				ContextName: "remoteAddress",
				Operator:    api.OperatorInCidr,
				Values:      []string{"160.33.0.0/16"},
			},
			expected: true,
		},
		{
			name: "cidr range no match",
			ctx:  &context.Context{RemoteAddress: "160.34.0.1"},
			constraint: api.Constraint{
				ContextName: "remoteAddress",
				Operator:    api.OperatorInCidr,
				Values:      []string{"160.33.0.0/16"},
			},
			expected: false,
		},
		{
			name: "ignore invalid values when valid exists",
			ctx:  &context.Context{RemoteAddress: "192.168.1.42"},
			constraint: api.Constraint{
				ContextName: "remoteAddress",
				Operator:    api.OperatorInCidr,
				Values:      []string{"127.invalid", "192.168.1.0/24"},
			},
			expected: true,
		},
		{
			name: "invalid-only values",
			ctx:  &context.Context{RemoteAddress: "127.0.0.1"},
			constraint: api.Constraint{
				ContextName: "remoteAddress",
				Operator:    api.OperatorInCidr,
				Values:      []string{"127.invalid"},
			},
			expected: false,
		},
		{
			name: "missing remote address",
			ctx:  &context.Context{},
			constraint: api.Constraint{
				ContextName: "remoteAddress",
				Operator:    api.OperatorInCidr,
				Values:      []string{"127.0.0.1"},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, operatorInCidr(tc.ctx, tc.constraint))
		})
	}
}
