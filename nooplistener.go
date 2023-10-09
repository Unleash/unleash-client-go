package unleash

// DebugListener is an implementation of all of the listener interfaces that simply logs
// debug info to stdout. It is meant for debugging purposes and an example of implementing
// the listener interfaces.
type NoopListener struct{}

/*
// OnError prints out errors.
func (l NoopListener) OnError(err error) {
}

// OnWarning prints out warning.
func (l NoopListener) OnWarning(warning error) {
}

// OnReady prints to the console when the repository is ready.
func (l NoopListener) OnReady() {
}

// OnCount prints to the console when the feature is queried.
func (l NoopListener) OnCount(name string, enabled bool) {
}
// OnSent prints to the console when the server has uploaded metrics.
func (l NoopListener) OnSent(payload MetricsData) {
}

// OnRegistered prints to the console when the client has registered.
func (l NoopListener) OnRegistered(payload ClientData) {
}
*/
