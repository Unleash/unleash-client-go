package main

import (
	"fmt"
	"github.com/unleash/unleash-client-go"
	"time"
)

func main() {
	client, err := unleash.NewClient(
		unleash.WithAppName("my-application"),
		unleash.WithUrl("http://unleash.herokuapp.com/api/"),
	)

	if err != nil {
		fmt.Printf("ERROR: Starting client: %v", err)
		return
	}

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
