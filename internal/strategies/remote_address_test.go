package strategies

import (
	"github.com/Unleash/unleash-client-go/v4/context"
	"github.com/Unleash/unleash-client-go/v4/strategy"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRemoteAddressStrategy_Name(t *testing.T) {
	strategy := NewRemoteAddressStrategy()
	assert.Equal(t, "remoteAddress", strategy.Name(), "strategy should have correct name")
}

func TestRemoteAddressStrategy_IsEnabled(t *testing.T) {
	s := NewRemoteAddressStrategy()
	assert := assert.New(t)

	t.Run("r=", func(t *testing.T) {
		var params map[string]interface{}
		ctx := &context.Context{
			RemoteAddress: "123",
		}
		assert.False(s.IsEnabled(params, ctx), "RemoteAddressStrategy should not crash for missing params")
	})

	t.Run("r=i", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamIps: "127.0.0.1",
		}
		ctx := &context.Context{
			RemoteAddress: "127.0.0.1",
		}
		assert.True(s.IsEnabled(params, ctx), "RemoteAddressStrategy should be enabled for ip")
	})

	t.Run("r!=list(i)", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamIps: "127.0.1.1, 127.0.1.2, 127.0.1.3",
		}
		ctx := &context.Context{
			RemoteAddress: "127.0.0.1",
		}
		assert.False(s.IsEnabled(params, ctx), "RemoteAddressStrategy should not be enabled for ip NOT in list")
	})

	t.Run("r=list(i)", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamIps: "127.0.0.1, 127.0.0.2,127.0.0.213",
		}
		ctx := &context.Context{
			RemoteAddress: "127.0.0.2",
		}
		assert.True(s.IsEnabled(params, ctx), "RemoteAddressStrategy should be enabled for ip in list")
	})

	t.Run("r=range(i)", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamIps: "127.0.1.1, 127.0.1.2,127.0.1.3, 160.33.0.0/16",
		}
		ctx := &context.Context{
			RemoteAddress: "160.33.0.33",
		}
		assert.True(s.IsEnabled(params, ctx), "RemoteAddressStrategy should be enabled for ip inside range in a list")
	})
}
