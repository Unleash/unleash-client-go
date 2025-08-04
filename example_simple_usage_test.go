package unleash_test

import (
	"fmt"
	"github.com/Unleash/unleash-client-go/v4"
	"time"
)

const PropertyName = "eid.enabled"

// ExampleSimpleUsage demonstrates the simplest way to use the unleash client.
func Example_simpleUsage() {
	unleash.Initialize(
		unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithAppName("my-application"),
		unleash.WithUrl("https://eu.app.unleash-hosted.com/demo/api/"),
	)

	timer := time.NewTimer(1 * time.Second)

	for {
		<-timer.C
		fmt.Printf("'%s' enabled? %v\n", PropertyName, unleash.IsEnabled(PropertyName))
		timer.Reset(1 * time.Second)
	}

}
