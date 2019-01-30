package feature

import (
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWithContext(t *testing.T) {
	ctx := context.Context{
		UserID: "123",
	}

	o := WithContext(ctx)
	opts := FlattenOptions(o)
	assert.Equal(t, &ctx, opts.Context())
}

func TestWithFallback(t *testing.T) {
	fallback := true

	o := WithFallback(fallback)
	opts := FlattenOptions(o)
	assert.Equal(t, &fallback, opts.Fallback())
}
