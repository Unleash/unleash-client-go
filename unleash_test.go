package unleash_test

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/Unleash/unleash-client-go/v4"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"
)

func Test_withVariants(t *testing.T) {
	a := assert.New(t)
	demoReader, err := os.Open("demo_app_toggles.json")
	defer gock.OffAll()

	gock.New("http://foo.com").
		Post("/client/register").
		Reply(200)
		// Read the file into a byte slice
	byteValue, _ := io.ReadAll(demoReader)
	// Convert the byte slice to a string
	jsonStr := string(byteValue)

	// Use the string as the body of the Gock request
	gock.New("http://foo.com").
		Get("/client/features").Reply(200).BodyString(jsonStr)
	if err != nil {
		t.Fail()
	}
	err = unleash.Initialize(
		unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithAppName("my-application"),
		unleash.WithRefreshInterval(5*time.Second),
		unleash.WithDisableMetrics(true),
		unleash.WithStorage(&unleash.BootstrapStorage{Reader: demoReader}),
		unleash.WithUrl("http://foo.com"),
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
	err = unleash.Close()
	a.Nil(err)
}

func Test_withVariantsAndANonExistingStrategyName(t *testing.T) {
	a := assert.New(t)
	demoReader, err := os.Open("demo_app_toggles.json")
	if err != nil {
		t.Fail()
	}
	defer gock.OffAll()

	gock.New("http://foo.com").
		Post("/client/register").
		Reply(200)

	byteValue, _ := io.ReadAll(demoReader)
	// Convert the byte slice to a string
	jsonStr := string(byteValue)

	// Use the string as the body of the Gock request
	gock.New("http://foo.com").
		Get("/client/features").Reply(200).BodyString(jsonStr)
	err = unleash.Initialize(
		unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithAppName("my-application"),
		unleash.WithRefreshInterval(20*time.Second),
		unleash.WithDisableMetrics(true),
		unleash.WithStorage(&unleash.BootstrapStorage{Reader: demoReader}),
		unleash.WithUrl("http://foo.com"),
	)

	if err != nil {
		t.Fail()
	}

	feature := unleash.GetVariant("AuditLog")
	fmt.Printf("feature is %v\n", feature)
	if feature.Enabled == true {
		t.Fatalf("Expected feature to be disabled because Environment does not exist as strategy")
	}
	err = unleash.Close()
	a.Nil(err)
}

func Test_IsEnabledWithUninitializedClient(t *testing.T) {
	result := unleash.IsEnabled("foo", unleash.WithFallback(true))
	if !result {
		t.Fatalf("Expected true")
	}

}
