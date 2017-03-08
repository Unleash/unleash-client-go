package strategies

import "testing"

func TestDefaultStrategy_Name(t *testing.T) {
	strategy := NewDefaultStrategy()

	if strategy.Name() != "default" {
		t.Errorf("strategy should have correct name: %s", strategy.Name())
	}
}

func TestDefaultStrategy_IsEnabled(t *testing.T) {
	s := NewDefaultStrategy()

	if !s.IsEnabled(nil, nil) {
		t.Errorf("default strategy should be enabled")
	}

}
