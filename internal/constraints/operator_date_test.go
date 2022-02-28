package constraints

import (
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestOperatorDate(t *testing.T) {
	testCases := []checkTestCase{
		{
			ctx:         &context.Context{CurrentTime: "2006-01-02T15:04:04Z"},
			constraints: []api.Constraint{{ContextName: "currentTime", Operator: "DATE_BEFORE", Value: "2006-01-02T15:04:05Z"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{CurrentTime: "2006-01-02T15:04:05Z"},
			constraints: []api.Constraint{{ContextName: "currentTime", Operator: "DATE_BEFORE", Value: "2006-01-02T15:04:05Z"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{CurrentTime: "2006-01-02T15:04:06Z"},
			constraints: []api.Constraint{{ContextName: "currentTime", Operator: "DATE_BEFORE", Value: "2006-01-02T15:04:05Z"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"date": "2006-01-02T15:04:04Z"}},
			constraints: []api.Constraint{{ContextName: "date", Operator: "DATE_BEFORE", Value: "2006-01-02T15:04:05Z"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"date": "2006-01-02T15:04:06Z"}},
			constraints: []api.Constraint{{ContextName: "date", Operator: "DATE_BEFORE", Value: "2006-01-02T15:04:05Z"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{},
			constraints: []api.Constraint{{ContextName: "currentTime", Operator: "DATE_AFTER", Value: time.Now().Add(-time.Minute).Format(time.RFC3339)}},
			expected:    true,
		},
		{
			ctx:         &context.Context{},
			constraints: []api.Constraint{{ContextName: "currentTime", Operator: "DATE_AFTER", Value: time.Now().Add(time.Minute).Format(time.RFC3339)}},
			expected:    false,
		},
		{
			ctx:         &context.Context{CurrentTime: "2006-01-02T15:04:04Z"},
			constraints: []api.Constraint{{ContextName: "currentTime", Operator: "DATE_AFTER", Value: "2006-01-02T15:04:05Z"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{CurrentTime: "2006-01-02T15:04:05Z"},
			constraints: []api.Constraint{{ContextName: "currentTime", Operator: "DATE_AFTER", Value: "2006-01-02T15:04:05Z"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{CurrentTime: "2006-01-02T15:04:06Z"},
			constraints: []api.Constraint{{ContextName: "currentTime", Operator: "DATE_AFTER", Value: "2006-01-02T15:04:05Z"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"date": "2006-01-02T15:04:05.001Z"}},
			constraints: []api.Constraint{{ContextName: "date", Operator: "DATE_AFTER", Value: "2006-01-02T15:04:05.002Z"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"date": "2006-01-02T15:04:05.001Z"}},
			constraints: []api.Constraint{{ContextName: "date", Operator: "DATE_AFTER", Value: "2006-01-02T15:04:05.001Z"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"date": "2006-01-02T15:04:05.002Z"}},
			constraints: []api.Constraint{{ContextName: "date", Operator: "DATE_AFTER", Value: "2006-01-02T15:04:05.001Z"}},
			expected:    true,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"date": "2006-01-02T15:04:05.999999999Z"}},
			constraints: []api.Constraint{{ContextName: "date", Operator: "DATE_AFTER", Value: "2006-01-02T15:04:05.999999999Z"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"date": "2006-01-02T15:04:05.999999998Z"}},
			constraints: []api.Constraint{{ContextName: "date", Operator: "DATE_AFTER", Value: "2006-01-02T15:04:05.999999999Z"}},
			expected:    false,
		},
		{
			ctx:         &context.Context{Properties: map[string]string{"date": "2006-01-02T15:04:05.999999999Z"}},
			constraints: []api.Constraint{{ContextName: "date", Operator: "DATE_AFTER", Value: "2006-01-02T15:04:05.999999998Z"}},
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
