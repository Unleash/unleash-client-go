package unleash

// NoopListener is an implementation of all of the listener interfaces that discards
// all messages. It's added if no other listener is added to drain the channels and as
// an example of implementing the listener interfaces.
type NoopListener struct{}

func (l NoopListener) OnError(err error) {
}

func (l NoopListener) OnWarning(warning error) {
}

// The repository is ready.
func (l NoopListener) OnReady() {
}

// The feature is queried.
func (l NoopListener) OnCount(name string, enabled bool) {
}

// The server has uploaded metrics.
func (l NoopListener) OnSent(payload MetricsData) {
}

// The client has registered.
func (l NoopListener) OnRegistered(payload ClientData) {
}
