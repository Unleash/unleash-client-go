package strategies

import (
	"github.com/Unleash/unleash-client-go/context"
	"github.com/Unleash/unleash-client-go/strategy"
	"testing"
)

func TestUserWithIdStrategy_Name(t *testing.T) {
	strategy := NewUserWithIdStrategy()

	if strategy.Name() != "userWithId" {
		t.Errorf("strategy should have correct name: %s", strategy.Name())
	}
}

func TestUserWithIdStrategy_IsEnabled(t *testing.T) {
	s := NewUserWithIdStrategy()

	t.Run("u=u", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamUserIds: "123",
		}
		ctx := &context.Context{
			UserId: "123",
		}

		if !s.IsEnabled(params, ctx) {
			t.Errorf("user-with-id-strategy should be enabled for userId")
		}
	})

	t.Run("u=list(a, u)", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamUserIds: "123, 122, 12312312",
		}
		ctx := &context.Context{
			UserId: "12312312",
		}

		if !s.IsEnabled(params, ctx) {
			t.Errorf("user-with-id-strategy should be enabled for userId in list")
		}
	})

	t.Run("u!=list(a, b)", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamUserIds: "123, 122, 122",
		}
		ctx := &context.Context{
			UserId: "12",
		}

		if s.IsEnabled(params, ctx) {
			t.Errorf("user-with-id-strategy should not be enabled for userId NOT in list")
		}
	})

	t.Run("u=list(a,u)", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamUserIds: "123,122,12312312",
		}
		ctx := &context.Context{
			UserId: "122",
		}

		if !s.IsEnabled(params, ctx) {
			t.Errorf("user-with-id-strategy should be enabled for userId in list")
		}
	})

}
