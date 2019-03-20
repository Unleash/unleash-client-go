package unleash

import (
	"log"
)

// DefaultListener is a listener that logs events using the standard logger.
type DefaultListener struct{}

// OnError prints out errors.
func (l DefaultListener) OnError(err error) {
	log.Printf("ERROR: %s\n", err.Error())
}

// OnWarning prints out warning.
func (l DefaultListener) OnWarning(warning error) {
	log.Printf("WARNING: %s\n", warning.Error())
}

// OnReady prints to the console when the repository is ready.
func (l DefaultListener) OnReady() {
	log.Printf("READY\n")
}

// OnCount prints to the console when the feature is queried.
func (l DefaultListener) OnCount(name string, enabled bool) {
	log.Printf("Counted '%s'  as enabled? %v\n", name, enabled)
}

// OnSent prints to the console when the server has uploaded metrics.
func (l DefaultListener) OnSent(payload MetricsData) {
	log.Printf("Sent: %+v\n", payload)
}

// OnRegistered prints to the console when the client has registered.
func (l DefaultListener) OnRegistered(payload ClientData) {
	log.Printf("Registered: %+v\n", payload)
}
