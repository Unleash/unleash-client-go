package strategies

import (
	"testing"
	"github.com/unleash/unleash-client-go/context"
	"os"
	"github.com/unleash/unleash-client-go/strategy"
)

func TestApplicationHostnameStrategy_Name(t *testing.T) {
	strategy := NewApplicationHostnameStrategy()

	if strategy.Name() != "applicationHostname" {
		t.Errorf("strategy should have correct name: %s", strategy.Name())
	}
}

func TestApplicationHostnameStrategy_IsEnabled(t *testing.T) {
	s := NewApplicationHostnameStrategy()

	t.Run("h=''", func(t *testing.T) {
		isEnabled := s.IsEnabled(nil, &context.Context{
			Properties: map[string]string{
				"hostNames": "",
			},
		})

		if isEnabled {
			t.Errorf("strategy should be disabled when no hostname defined")
		}
	})

	t.Run("h=os.hostname", func(t *testing.T) {
		hostname, _ := os.Hostname()
		isEnabled := s.IsEnabled(map[string]interface{}{
			strategy.ParamHostNames: hostname,
		}, nil)

		if !isEnabled {
			t.Errorf("strategy should be enabled when hostname is defined")
		}
	})

	t.Run("h=list(os.hostname)", func(t *testing.T) {
		hostname, _ := os.Hostname()
		isEnabled := s.IsEnabled(map[string]interface{}{
			strategy.ParamHostNames: "localhost," + hostname,
		}, nil)

		if !isEnabled {
			t.Errorf("strategy should be enabled when hostname is defined in list")
		}
	})

	t.Run("h=list(a,env.hostname)", func(t *testing.T) {
		os.Setenv("HOSTNAME", "some-random-name")

		// needed to re-read env-var
		s = NewApplicationHostnameStrategy()

		isEnabled := s.IsEnabled(map[string]interface{}{
			strategy.ParamHostNames: "localhost,some-random-name",
		}, nil)

		if !isEnabled {
			t.Errorf("strategy should be enabled when hostname is defined via env")
		}
	})

	t.Run("h=list(a,env.hostNAME)", func(t *testing.T) {
		os.Setenv("HOSTNAME", "some-random-NAME")

		// needed to re-read env-var
		s = NewApplicationHostnameStrategy()

		isEnabled := s.IsEnabled(map[string]interface{}{
			strategy.ParamHostNames: "localhost,some-random-name",
		}, nil)

		if !isEnabled {
			t.Errorf("strategy should handle wierd casing")
		}
	})
}

