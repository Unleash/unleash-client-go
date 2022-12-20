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

	feature := unleash.GetVariant("Demo")
	fmt.Printf("feature is %v\n", feature)
	if feature.Enabled == false {
		t.Fatalf("Expected feature to be enabled")
	}
	if feature.Name != "small" && feature.Name != "medium" {
		t.Fatalf("Expected one of the variant names")
	}
	if feature.Payload.Value != "35" && feature.Payload.Value != "55" {
		t.Fatalf("Expected one of the variant payloads")
	}
}

func Test_withVariantsAndUnexistentUnexistingStrategy(t *testing.T) {
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

	feature := unleash.GetVariant("AuditLog")
	fmt.Printf("feature is %v\n", feature)
	if feature.Enabled == true {
		t.Fatalf("Expected feature to be disabled because Environment does not exist as strategy")
	}
}
