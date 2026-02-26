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

// quick and dirty to get a sense of the perf hit here
func BenchmarkCheck_OperatorRegex(b *testing.B) {
	ctx := &context.Context{Properties: map[string]string{"email": "example@getunleash.ai"}}
	constraints := []api.Constraint{{ContextName: "email", Operator: "REGEX", Value: ".*@getunleash\\.ai$"}}

	// warm up, not actually sure if go bench will do this for us but this is relatively cheap and I trust nothing at this point
	_, _ = Check(ctx, constraints)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ok, err := Check(ctx, constraints)
		if err != nil || !ok {
			b.Fatalf("unexpected result ok=%v err=%v", ok, err)
		}
	}
}
