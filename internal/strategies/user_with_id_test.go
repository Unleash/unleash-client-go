package strategies

import (
	"github.com/Unleash/unleash-client-go/context"
	"github.com/Unleash/unleash-client-go/strategy"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUserWithIdStrategy_Name(t *testing.T) {
	strategy := NewUserWithIdStrategy()
	assert.Equal(t, "userWithId", strategy.Name(), "strategy should have correct name")
}

func TestUserWithIdStrategy_IsEnabled(t *testing.T) {
	var s strategy.Strategy
	s = NewUserWithIdStrategy()
	assert := assert.New(t)

	t.Run("u=u", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamUserIds: "123",
		}

		as := s
		if adoptable, ok := s.(strategy.AdoptableStrategy); ok {
			as = adoptable.Adopt(params)
		}

		ctx := &context.Context{
			UserId: "123",
		}
		assert.True(as.IsEnabled(params, ctx), "user-with-id-strategy should be enabled for userId")
	})

	t.Run("u=list(a, u)", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamUserIds: "123, 122, 12312312",
		}

		as := s
		if adoptable, ok := s.(strategy.AdoptableStrategy); ok {
			as = adoptable.Adopt(params)
		}

		ctx := &context.Context{
			UserId: "12312312",
		}
		assert.True(as.IsEnabled(params, ctx), "user-with-id-strategy should be enabled for userId in list")
	})

	t.Run("u!=list(a, b)", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamUserIds: "123, 122, 122",
		}

		as := s
		if adoptable, ok := s.(strategy.AdoptableStrategy); ok {
			as = adoptable.Adopt(params)
		}

		ctx := &context.Context{
			UserId: "12",
		}
		assert.False(as.IsEnabled(params, ctx), "user-with-id-strategy should not be enabled for userId NOT in list")
	})

	t.Run("u=list(a,u)", func(t *testing.T) {
		params := map[string]interface{}{
			strategy.ParamUserIds: "123,122,12312312",
		}

		as := s
		if adoptable, ok := s.(strategy.AdoptableStrategy); ok {
			as = adoptable.Adopt(params)
		}

		ctx := &context.Context{
			UserId: "122",
		}
		assert.True(as.IsEnabled(params, ctx), "user-with-id-strategy should be enabled for userId in list")
	})
}
