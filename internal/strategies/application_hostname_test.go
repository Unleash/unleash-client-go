package strategies

import (
	"github.com/Unleash/unleash-client-go/context"
	"github.com/Unleash/unleash-client-go/strategy"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestApplicationHostnameStrategy_Name(t *testing.T) {
	strategy := NewApplicationHostnameStrategy()
	assert.Equal(t, "applicationHostname", strategy.Name(), "strategy should have correct name")
}

func TestApplicationHostnameStrategy_IsEnabled(t *testing.T) {
	s := NewApplicationHostnameStrategy()
	assert := assert.New(t)

	t.Run("h=''", func(t *testing.T) {
		isEnabled := s.IsEnabled(nil, &context.Context{
			Properties: map[string]string{
				"hostNames": "",
			},
		})

		assert.False(isEnabled, "strategy should be disabled when no hostname defined")
	})

	t.Run("h=os.hostname", func(t *testing.T) {
		hostname, _ := resolveHostname()
		isEnabled := s.IsEnabled(map[string]interface{}{
			strategy.ParamHostNames: hostname,
		}, nil)

		assert.True(isEnabled, "strategy should be enabled when hostname is defined")
	})

	t.Run("h=list(os.hostname)", func(t *testing.T) {
		hostname, _ := resolveHostname()
		isEnabled := s.IsEnabled(map[string]interface{}{
			strategy.ParamHostNames: "localhost," + hostname,
		}, nil)

		assert.True(isEnabled, "strategy should be enabled when hostname is defined in list")
	})

	t.Run("h=list(a,env.hostname)", func(t *testing.T) {
		os.Setenv("HOSTNAME", "some-random-name")

		// needed to re-read env-var
		s = NewApplicationHostnameStrategy()

		isEnabled := s.IsEnabled(map[string]interface{}{
			strategy.ParamHostNames: "localhost,some-random-name",
		}, nil)

		assert.True(isEnabled, "strategy should be enabled when hostname is defined via env")
	})

	t.Run("h=list(a,env.hostNAME)", func(t *testing.T) {
		os.Setenv("HOSTNAME", "some-random-NAME")

		// needed to re-read env-var
		s = NewApplicationHostnameStrategy()

		isEnabled := s.IsEnabled(map[string]interface{}{
			strategy.ParamHostNames: "localhost,some-random-name",
		}, nil)

		assert.True(isEnabled, "strategy should handle weird casing")
	})
}
