package unleash_test

import (
	"fmt"
	"time"

	"github.com/Unleash/unleash-go-sdk/v5"
)

func ExampleClient_streamingMode() {
	// Initialize the Unleash client with streaming mode enabled
	client, err := unleash.NewClient(
		unleash.WithUrl("https://unleash.example.com"),
		unleash.WithAppName("my-application"),
		unleash.WithEnvironment("production"),
		// Enable experimental streaming mode for real-time updates
		unleash.WithExperimentalMode(map[string]string{
			"type": "streaming",
		}),
		// Optional: Set a custom refresh interval as fallback
		unleash.WithRefreshInterval(15*time.Second),
		// Add a listener to get notified when updates arrive
		unleash.WithListener(&unleash.DebugListener{}),
	)

	if err != nil {
		fmt.Printf("Failed to initialize client: %v\n", err)
		return
	}
	defer client.Close()

	// Wait for the client to be ready (initial features loaded via streaming)
	client.WaitForReady()

	// Now the client will receive real-time updates via Server-Sent Events
	// Check if a feature is enabled
	isEnabled := client.IsEnabled("my-feature")
	fmt.Printf("Feature 'my-feature' is enabled: %v\n", isEnabled)

	// The client will automatically receive updates when features change
	// on the server without polling delays
}

func ExampleClient_streamingWithFallback() {
	// Streaming mode with automatic fallback to polling
	client, err := unleash.NewClient(
		unleash.WithUrl("https://unleash.example.com"),
		unleash.WithAppName("my-application"),
		// Enable streaming mode
		unleash.WithExperimentalMode(map[string]string{
			"type": "streaming",
		}),
		// If streaming fails, the client will automatically fall back to polling
		// with this refresh interval
		unleash.WithRefreshInterval(30*time.Second),
	)

	if err != nil {
		fmt.Printf("Failed to initialize client: %v\n", err)
		return
	}
	defer client.Close()

	// The client will try to establish a streaming connection
	// If it fails, it will automatically fall back to polling mode
	client.WaitForReady()

	// Use the client normally - it will work with either streaming or polling
	isEnabled := client.IsEnabled("my-feature")
	fmt.Printf("Feature is enabled: %v\n", isEnabled)
}