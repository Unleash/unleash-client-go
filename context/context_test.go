package context

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOverride(t *testing.T) {
	type testCase struct {
		input    *Context
		src      *Context
		expected *Context
	}

	testCases := []testCase{
		{
			input: &Context{
				AppName:     "testApp",
				Environment: "production",
			},
			src: &Context{
				UserId:        "12345",
				SessionId:     "ABCDE",
				RemoteAddress: "127.0.0.1",
				Properties: map[string]string{
					"IsAdmin": "true",
				},
			},
			expected: &Context{
				UserId:        "12345",
				SessionId:     "ABCDE",
				RemoteAddress: "127.0.0.1",
				Environment:   "production",
				AppName:       "testApp",
				Properties: map[string]string{
					"IsAdmin": "true",
				},
			},
		},
	}

	for _, tc := range testCases {
		actual := tc.input.Override(*tc.src)
		assert.Equal(t, tc.expected, actual)
	}
}

func TestContext_Field(t *testing.T) {
	ctx := Context{
		UserId:        "userId",
		SessionId:     "sessionId",
		RemoteAddress: "remoteAddress",
		Environment:   "environment",
		AppName:       "appName",
		Properties: map[string]string{
			"Foo": "Bar",
		},
	}

	assert.Equal(t, ctx.UserId, ctx.Field(ctx.UserId))
	assert.Equal(t, ctx.SessionId, ctx.Field(ctx.SessionId))
	assert.Equal(t, ctx.RemoteAddress, ctx.Field(ctx.RemoteAddress))
	assert.Equal(t, ctx.Environment, ctx.Field(ctx.Environment))
	assert.Equal(t, ctx.AppName, ctx.Field(ctx.AppName))
	assert.Equal(t, "Bar", ctx.Field("Foo"))
}
