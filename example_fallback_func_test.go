package unleash_test

import (
	"fmt"
	"github.com/konfortes/unleash-client-go/v3"
	"github.com/konfortes/unleash-client-go/v3/context"
	"time"
)

const MissingFeature = "does_not_exist"

// ExampleFallbackFunc demonstrates how to specify a fallback function.
func Example_fallbackFunc() {
	unleash.Initialize(
		unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithAppName("my-application"),
		unleash.WithUrl("http://unleash.herokuapp.com/api/"),
	)

	fallback := func(feature string, ctx *context.Context) bool {
		return feature == MissingFeature
	}

	timer := time.NewTimer(1 * time.Second)

	for {
		<-timer.C
		isEnabled := unleash.IsEnabled(MissingFeature, unleash.WithFallbackFunc(fallback))
		fmt.Printf("'%s' enabled? %v\n", PropertyName, isEnabled)
		timer.Reset(1 * time.Second)
	}
}
