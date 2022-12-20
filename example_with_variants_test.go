package unleash_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Unleash/unleash-client-go/v3"
)

func Test_withVariants(t *testing.T) {
	demoReader, err := os.Open("demo_app_toggles.json")
	if err != nil {
		t.Fail()
	}
	err = unleash.Initialize(
		unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithAppName("my-application"),
		unleash.WithRefreshInterval(5*time.Second),
		unleash.WithMetricsInterval(5*time.Second),
		unleash.WithStorage(&unleash.BootstrapStorage{Reader: demoReader}),
		unleash.WithUrl("https://localhost:4242"),
	)

	if err != nil {
		t.Fail()
	}

	enabled := unleash.GetVariant("ActivityLog")
	fmt.Printf("feature is enabled? %v\n", enabled.Enabled)
	if enabled.Enabled == true {
		t.Fatalf("Expected feature to be enabled")
	}
}
