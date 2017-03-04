package unleash

import (
	"fmt"
)

// DebugListener is an implementation of all of the listener interfaces that simply logs
// debug info to stdout. It is meant for debugging purposes and an example of implementing
// the listener interfaces.
type DebugListener struct{}

// OnError prints out errors.
func (l DebugListener) OnError(err error) {
	fmt.Printf("ERROR: %s\n", err.Error())
}

// OnWarning prints out warning.
func (l DebugListener) OnWarning(warning error) {
	fmt.Printf("WARNING: %s\n", warning.Error())
}

// OnReady prints to the console when the repository is ready.
func (l DebugListener) OnReady() {
	fmt.Printf("READY\n")
}

// OnCount prints to the console when the feature is queried.
func (l DebugListener) OnCount(name string, enabled bool) {
	fmt.Printf("Counted '%s'  as enabled? %v\n", name, enabled)
}

// OnSent prints to the console when the server has uploaded metrics.
func (l DebugListener) OnSent(payload MetricsData) {
	fmt.Printf("Sent: %+v\n", payload)
}

// OnRegistered prints to the console when the client has registered.
func (l DebugListener) OnRegistered(payload ClientData) {
	fmt.Printf("Registered: %+v\n", payload)
}
