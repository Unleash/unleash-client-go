package unleash

import "github.com/Unleash/unleash-go-sdk/v6/api"

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

	// OnUpdate is called each time the client has loaded the features toggles from
	// the Unleash server after the initial request.
	OnUpdate()
}

// ImpressionListener defines an interface that can be implemented in order to receive
// impression events when a feature with impression data is evaluated.
type ImpressionListener interface {
	// OnImpression is called whenever a feature with impression data is evaluated.
	OnImpression(ImpressionEvent)
}

// IsEnabled queries the default client whether or not the specified feature is enabled or not.
func IsEnabled(feature string, options FeatureOptions) bool {
	if defaultClient == nil {
		return handleFallback(options.FallbackFunc, options.Fallback, feature, &options.Ctx).Enabled
	}
	return defaultClient.IsEnabled(feature, options)
}

// Initialize will specify the options to be used by the default client.
func Initialize(options ...ConfigOption) (err error) {
	defaultClient, err = NewClient(options...)
	return err
}

func GetVariant(feature string, options VariantOptions) *api.Variant {
	if defaultClient == nil {
		return api.GetDefaultVariant()
	}
	return defaultClient.GetVariant(feature, options)
}

// Close will close the default client.
func Close() error {
	if defaultClient == nil {
		return nil
	}
	return defaultClient.Close()
}

// WaitForReady will block until the default client is ready or return immediately.
func WaitForReady() {
	defaultClient.WaitForReady()
}

// DefineCounter registers a counter metric for impact metrics reporting.
func DefineCounter(name, help string) {
	if defaultClient == nil {
		return
	}
	defaultClient.DefineCounter(name, help)
}

// IncrementCounter increments a previously defined counter metric.
func IncrementCounter(name string) {
	if defaultClient == nil {
		return
	}
	defaultClient.IncrementCounter(name)
}

// IncrementCounterBy increments a previously defined counter metric by a specified value.
func IncrementCounterBy(name string, value int64) {
	if defaultClient == nil {
		return
	}
	defaultClient.IncrementCounterBy(name, value)
}

// DefineGauge registers a gauge metric for impact metrics reporting.
func DefineGauge(name, help string) {
	if defaultClient == nil {
		return
	}
	defaultClient.DefineGauge(name, help)
}

// UpdateGauge updates the value for a previously defined gauge metric.
func UpdateGauge(name string, value float64) {
	if defaultClient == nil {
		return
	}
	defaultClient.UpdateGauge(name, value)
}

// DefineHistogram registers a histogram metric for impact metrics reporting.
func DefineHistogram(name, help string, buckets ...float64) {
	if defaultClient == nil {
		return
	}
	defaultClient.DefineHistogram(name, help, buckets...)
}

// ObserveHistogram records a value in a previously defined histogram metric.
func ObserveHistogram(name string, value float64) {
	if defaultClient == nil {
		return
	}
	defaultClient.ObserveHistogram(name, value)
}
