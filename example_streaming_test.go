package unleash_test

import (
	"fmt"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5"
)

func ExampleClient_streamingMode() {
	client, err := unleash.NewClient(
		unleash.WithUrl("https://unleash.example.com"),
		unleash.WithAppName("my-application"),
		unleash.WithEnvironment("production"),
		unleash.WithExperimentalMode(map[string]string{
			"type": "streaming",
		}),
		unleash.WithRefreshInterval(15*time.Second),
		unleash.WithListener(&unleash.DebugListener{}),
	)

	if err != nil {
		fmt.Printf("Failed to initialize client: %v\n", err)
		return
	}
	defer client.Close()

	client.WaitForReady()

	isEnabled := client.IsEnabled("my-feature")
	fmt.Printf("Feature 'my-feature' is enabled: %v\n", isEnabled)
}

func ExampleClient_streamingWithFallback() {
	client, err := unleash.NewClient(
		unleash.WithUrl("https://unleash.example.com"),
		unleash.WithAppName("my-application"),
		unleash.WithExperimentalMode(map[string]string{
			"type": "streaming",
		}),
		unleash.WithRefreshInterval(30*time.Second),
	)

	if err != nil {
		fmt.Printf("Failed to initialize client: %v\n", err)
		return
	}
	defer client.Close()

	client.WaitForReady()

	isEnabled := client.IsEnabled("my-feature")
	fmt.Printf("Feature is enabled: %v\n", isEnabled)
}