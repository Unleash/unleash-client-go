package unleash

import (
	"fmt"
)

type DebugListener struct{}

func (l DebugListener) OnError(err error) {
	fmt.Printf("ERROR: %s\n", err.Error())
}

func (l DebugListener) OnWarning(warning error) {
	fmt.Printf("WARNING: %s\n", warning.Error())
}

func (l DebugListener) OnReady() {
	fmt.Printf("READY\n")
}

func (l DebugListener) OnCount(name string, enabled bool) {
	fmt.Printf("Counted '%s'  as enabled? %v\n", name, enabled)
}

func (l DebugListener) OnSent(payload MetricsData) {
	fmt.Printf("Sent: %+v\n", payload)
}

func (l DebugListener) OnRegistered(payload ClientData) {
	fmt.Printf("Registered: %+v\n", payload)
}
