package unleash_test

import (
	"fmt"
	"github.com/Unleash/unleash-client-go/v4"
	"github.com/Unleash/unleash-client-go/v4/context"
	"strings"
	"time"
)

type ActiveForUserWithEmailStrategy struct{}

func (s ActiveForUserWithEmailStrategy) Name() string {
	return "ActiveForUserWithEmail"
}

func (s ActiveForUserWithEmailStrategy) IsEnabled(params map[string]interface{}, ctx *context.Context) bool {

	if ctx == nil {
		return false
	}
	value, found := params["emails"]
	if !found {
		return false
	}

	emails, ok := value.(string)
	if !ok {
		return false
	}

	for _, e := range strings.Split(emails, ",") {
		if e == ctx.Properties["emails"] {
			return true
		}
	}

	return false
}

// ExampleCustomStrategy demonstrates using a custom strategy.
func Example_customStrategy() {
	unleash.Initialize(
		unleash.WithListener(&unleash.DebugListener{}),
		unleash.WithAppName("my-application"),
		unleash.WithUrl("https://unleash.herokuapp.com/api/"),
		unleash.WithRefreshInterval(5*time.Second),
		unleash.WithMetricsInterval(5*time.Second),
		unleash.WithStrategies(&ActiveForUserWithEmailStrategy{}),
	)

	ctx := context.Context{
		Properties: map[string]string{
			"emails": "example@example.com",
		},
	}

	timer := time.NewTimer(1 * time.Second)

	for {
		<-timer.C
		enabled := unleash.IsEnabled("unleash.me", unleash.WithContext(ctx))
		fmt.Printf("feature is enabled? %v\n", enabled)
		timer.Reset(1 * time.Second)
	}

}
