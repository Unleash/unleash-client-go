package unleash

import "github.com/Unleash/unleash-client-go/v3/api"

var defaultClient *Client

// ErrorListener defines an interface that be implemented in order to receive
// errors and warnings from the client.
type ErrorListener interface {
	// OnError is called whenever the client experiences an error.
	OnError(error)

	// OnWarning is called whenever the client experiences a warning.
	OnWarning(error)
}

// MetricListener defines an interface that can be implemented in order to receive
// events that are relevant to sending metrics.
type MetricListener interface {
	// OnCount is called whenever the specified feature is queried.
	OnCount(string, bool)

	// OnSent is called whenever the server has successfully sent metrics to the server.
	OnSent(MetricsData)

	// OnRegistered is called whenever the client has successfully registered with the metrics server.
	OnRegistered(ClientData)
}

// RepositoryListener defines an interface that can be implemented in order to receive events that are relevant to
// the feature toggle repository.
type RepositoryListener interface {
	// OnReady is called when the client has loaded the feature toggles from
	// the Unleash server.
	OnReady()
}

// IsEnabled queries the default client whether or not the specified feature is enabled or not.
func IsEnabled(feature string, options ...FeatureOption) bool {
	return defaultClient.IsEnabled(feature, options...)
}

// Initialize will specify the options to be used by the default client.
func Initialize(options ...ConfigOption) (err error) {
	defaultClient, err = NewClient(options...)
	return
}

func GetVariant(feature string, options ...VariantOption) *api.Variant {
	return defaultClient.GetVariant(feature, options...)
}

// Close will close the default client.
func Close() error {
	return defaultClient.Close()
}

// WaitForReady will block until the default client is ready or return immediately.
func WaitForReady() {
	defaultClient.WaitForReady()
}
