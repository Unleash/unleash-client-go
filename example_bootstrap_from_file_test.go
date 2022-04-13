package unleash_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Unleash/unleash-client-go/v3"
	"github.com/Unleash/unleash-client-go/v3/context"
)

func Test_bootstrapFromFile(t *testing.T) {
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

	enabled := unleash.IsEnabled("DateExample", unleash.WithContext(context.Context{}))
	fmt.Printf("feature is enabled? %v\n", enabled)
	if enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}
}
