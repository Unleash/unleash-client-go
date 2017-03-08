package strategies

import (
	"github.com/Unleash/unleash-client-go/context"
	"github.com/Unleash/unleash-client-go/strategy"
	"testing"
)

func TestRemoteAddressStrategy_Name(t *testing.T) {
	strategy := NewRemoteAddressStrategy()

	if strategy.Name() != "remoteAddress" {
		t.Errorf("strategy should have correct name: %s", strategy.Name())
	}
}

func TestRemoteAddressStrategy_IsEnabled(t *testing.T) {
	s := NewRemoteAddressStrategy()

	t.Run("r=", func(t *testing.T) {
		var params map[string]interface{}
		ctx := &context.Context{
			RemoteAddress: "123",
		}

		if s.IsEnabled(params, ctx) {
			t.Errorf("RemoteAddressStrategy should not crash for missing params")
		}
	})

	t.Run("r=i", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamIps: "127.0.0.1",
		}
		ctx := &context.Context{
			RemoteAddress: "127.0.0.1",
		}

		if !s.IsEnabled(params, ctx) {
			t.Errorf("RemoteAddressStrategy should be enabled for ip")
		}
	})

	t.Run("r!=list(i)", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamIps: "127.0.1.1, 127.0.1.2, 127.0.1.3",
		}
		ctx := &context.Context{
			RemoteAddress: "127.0.0.1",
		}

		if s.IsEnabled(params, ctx) {
			t.Errorf("RemoteAddressStrategy should not be enabled for ip NOT in list")
		}
	})

	t.Run("r=list(i)", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamIps: "127.0.0.1, 127.0.0.2,127.0.0.213",
		}
		ctx := &context.Context{
			RemoteAddress: "127.0.0.2",
		}

		if !s.IsEnabled(params, ctx) {
			t.Errorf("RemoteAddressStrategy should be enabled for ip in list")
		}
	})
}
