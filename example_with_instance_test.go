package unleash_test

import (
	"fmt"
	"github.com/Unleash/unleash-client-go/v3"
	"time"
)

// Sync runs the client event loop. All of the channels must be read to avoid blocking the
// client.
func Sync(client *unleash.Client) {
	timer := time.NewTimer(1 * time.Second)
	for {
		select {
		case e := <-client.Errors():
			fmt.Printf("ERROR: %v\n", e)
		case w := <-client.Warnings():
			fmt.Printf("WARNING: %v\n", w)
		case <-client.Ready():
			fmt.Printf("READY\n")
		case m := <-client.Count():
			fmt.Printf("COUNT: %+v\n", m)
		case md := <-client.Sent():
			fmt.Printf("SENT: %+v\n", md)
		case cd := <-client.Registered():
			fmt.Printf("REGISTERED: %+v\n", cd)
		case <-timer.C:
			fmt.Printf("ISENABLED: %v\n", client.IsEnabled("eid.enabled"))
			timer.Reset(1 * time.Second)
		}
	}
}

// ExampleWithInstance demonstrates how to create the client manually instead of using the default client.
// It also shows how to run the event loop manually.
func Example_withInstance() {

	// Create the client with the desired options
	client, err := unleash.NewClient(
		unleash.WithListener(nil),
		unleash.WithAppName("my-application"),
		unleash.WithUrl("http://unleash.herokuapp.com/api/"),
	)

	if err != nil {
		fmt.Printf("ERROR: Starting client: %v", err)
		return
	}

	Sync(client)
}
