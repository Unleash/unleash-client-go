package unleash

var defaultClient *Client

type ErrorListener interface {
	OnError(error)
	OnWarning(error)
}

type MetricListener interface {
	OnCount(string, bool)
	OnSent(MetricsData)
	OnRegistered(ClientData)
}

type RepositoryListener interface {
	OnReady()
}

func IsEnabled(feature string, options ...FeatureOption) bool {
	return defaultClient.IsEnabled(feature, options...)
}

func Initialize(options ...ConfigOption) (err error) {
	defaultClient, err = NewClient(options...)
	return
}

func Close() error {
	return defaultClient.Close()
}
