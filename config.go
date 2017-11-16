package unleash

import (
	"net/http"
	"net/url"
	"time"

	"github.com/Unleash/unleash-client-go/context"
	"github.com/Unleash/unleash-client-go/strategy"
)

type configOption struct {
	appName         string
	instanceId      string
	url             string
	refreshInterval time.Duration
	metricsInterval time.Duration
	disableMetrics  bool
	backupPath      string
	strategies      []strategy.Strategy
	listener        interface{}
	storage         Storage
	httpClient      *http.Client
	customHeaders   http.Header
}

// ConfigOption represents a option for configuring the client.
type ConfigOption func(*configOption)

// WithListener allows users to register a type that implements one or more of
// the listener interfaces. If no listener is registered then the user is responsible
// for draining the various channels on the client. Failure to do so will stop the client
// from working as the worker routines will be blocked.
func WithListener(listener interface{}) ConfigOption {
	return func(o *configOption) {
		o.listener = listener
	}
}

// WithAppName specifies the name of the application.
func WithAppName(appName string) ConfigOption {
	return func(o *configOption) {
		o.appName = appName
	}
}

// WithInstanceId specifies the instance identifier of the current instance. If not provided,
// one will be generated based on various parameters such as current user and hostname.
func WithInstanceId(instanceId string) ConfigOption {
	return func(o *configOption) {
		o.instanceId = instanceId
	}
}

// WithUrl specifies the url of the unleash server the user is connecting to.
func WithUrl(url string) ConfigOption {
	return func(o *configOption) {
		o.url = url
	}
}

// WithRefreshInterval specifies the time interval with which the client should sync the
// feature toggles from the unleash server.
func WithRefreshInterval(refreshInterval time.Duration) ConfigOption {
	return func(o *configOption) {
		o.refreshInterval = refreshInterval
	}
}

// WithMetricsInterval specifies the times interval woth which the client should upload
// the metrics data to the unleash server.
func WithMetricsInterval(metricsInterval time.Duration) ConfigOption {
	return func(o *configOption) {
		o.metricsInterval = metricsInterval
	}
}

// WithDisabledMetrics specifies that the client should not log metrics to the unleash server.
func WithDisableMetrics(disableMetrics bool) ConfigOption {
	return func(o *configOption) {
		o.disableMetrics = disableMetrics
	}
}

// WithBackupPath specifies the path that is passed to the storage implementation for storing
// the feature toggles locally.
func WithBackupPath(backupPath string) ConfigOption {
	return func(o *configOption) {
		o.backupPath = backupPath
	}
}

// WithStrategies specifies which strategies (in addition to the defaults) should be used by the
// client.
func WithStrategies(strategies ...strategy.Strategy) ConfigOption {
	return func(o *configOption) {
		o.strategies = strategies
	}
}

// WithStorage specifies which storage implementation the repository should use for storing feature
// toggles.
func WithStorage(storage Storage) ConfigOption {
	return func(o *configOption) {
		o.storage = storage
	}
}

// WithHttpClient specifies which HttpClient the client should use for making requests to the server.
func WithHttpClient(client *http.Client) ConfigOption {
	return func(o *configOption) {
		o.httpClient = client
	}
}

// WithCustomHeaders specifies any custom headers that should be sent along with requests to the
// server.
func WithCustomHeaders(headers http.Header) ConfigOption {
	return func(o *configOption) {
		o.customHeaders = headers
	}
}

type featureOption struct {
	fallback *bool
	ctx      *context.Context
}

// FeatureOption provides options for querying if a feature is enabled or not.
type FeatureOption func(*featureOption)

// WithFallback specfies what the value should be if the feature toggle is not found on the
// unleash service.
func WithFallback(fallback bool) FeatureOption {
	return func(opts *featureOption) {
		opts.fallback = &fallback
	}
}

// WithContext allows the user to provide a context that will be passed into the active strategy
// for determining if a specified feature should be enabled or not.
func WithContext(ctx context.Context) FeatureOption {
	return func(opts *featureOption) {
		opts.ctx = &ctx
	}
}

type repositoryOptions struct {
	appName         string
	instanceId      string
	url             url.URL
	backupPath      string
	refreshInterval time.Duration
	storage         Storage
	httpClient      *http.Client
	customHeaders   http.Header
}

type metricsOptions struct {
	appName         string
	instanceId      string
	url             url.URL
	strategies      []string
	metricsInterval time.Duration
	disableMetrics  bool
	httpClient      *http.Client
	customHeaders   http.Header
}
