package unleash_client_go

var defaultClient *Client

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

func Warnings() <-chan error {
	return defaultClient.Warnings()
}

func Errors() <-chan error {
	return defaultClient.Errors()
}
