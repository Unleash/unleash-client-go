package main

import (
	"fmt"
	unleash "github.com/unleash/unleash-client-go"
	"time"
)

func init() {
	unleash.Initialize(
		unleash.WithAppName("my-application"),
		unleash.WithUrl("http://unleash.herokuapp.com/api/"),
	)
}

func main() {

	timer := time.NewTimer(1 * time.Second)

	for {
		select {
		case warning := <-unleash.Warnings():
			fmt.Printf("WARNING: %s", warning.Error())
		case err := <-unleash.Errors():
			fmt.Printf("ERROR: %s", err.Error())
		case <-timer.C:
			fmt.Printf("featureX is enabled? %v\n", unleash.IsEnabled("eid.enabled"))
			timer.Reset(1 * time.Second)
		}
	}

}
