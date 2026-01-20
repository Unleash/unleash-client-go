package unleash

import (
	"net/http"
	"net/url"
	"time"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/Unleash/unleash-go-sdk/v6/context"
	"github.com/Unleash/unleash-go-sdk/v6/strategy"
)

type configOption struct {
	appName          string
	environment      string
	instanceId       string
	url              string
	projectName      string
	refreshInterval  time.Duration
	metricsInterval  time.Duration
	disableMetrics   bool
	backupPath       string
	strategies       []strategy.Strategy
	listener         any
	storage          Storage
	httpClient       *http.Client
	customHeaders    http.Header
	experimentalMode map[string]string
}

// ConfigOption represents a option for configuring the client.
type ConfigOption func(*configOption)

// WithListener allows users to register a type that implements one or more of
// the listener interfaces. If no listener is registered then the user is responsible
// for draining the various channels on the client. Failure to do so will stop the client
// from working as the worker routines will be blocked.
func WithListener(listener any) ConfigOption {
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

// WithEnvironment specifies the environment
func WithEnvironment(env string) ConfigOption {
	return func(o *configOption) {
		o.environment = env
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
// feature toggles from the unleash server (default 15s).
func WithRefreshInterval(refreshInterval time.Duration) ConfigOption {
	return func(o *configOption) {
		o.refreshInterval = refreshInterval
	}
}

// WithMetricsInterval specifies the time interval with which the client should upload
// the metrics data to the unleash server (default 60s).
func WithMetricsInterval(metricsInterval time.Duration) ConfigOption {
	return func(o *configOption) {
		o.metricsInterval = metricsInterval
	}
}

// WithDisableMetrics specifies that the client should not log metrics to the unleash server.
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

// WithProjectName defines a projectName on the config object and is used to
// filter toggles by project name.
func WithProjectName(projectName string) ConfigOption {
	return func(o *configOption) {
		o.projectName = projectName
	}
}

// WithExperimentalMode allows enabling experimental features like streaming.
// Example: WithExperimentalMode(map[string]string{"type": "streaming"})
func WithExperimentalMode(mode map[string]string) ConfigOption {
	return func(o *configOption) {
		o.experimentalMode = mode
	}
}

// IsStreamingMode checks if streaming mode is configured
func (o *configOption) IsStreamingMode() bool {
	return o.experimentalMode != nil && o.experimentalMode["type"] == "streaming"
}

// FeatureResolver represents a function to be called to resolve the feature instead of using the repository
type FeatureResolver func(feature string) *api.Feature

// FallbackFunc represents a function to be called if the feature is not found.
type FallbackFunc func(feature string, ctx *context.Context) bool

// FeatureOptions controls how IsEnabled evaluates a feature toggle.
//
// Resolver:
//   - If Resolver is non-nil, it is used to resolve the feature by name instead of the repository.
//     This bypasses the normal resolution path and should no longer be necessary in v6.
//     It exists for backwards compatibility and may be removed in a future release.
//
// Context:
//   - If Ctx is non-nil, it is merged with the client's static context and used when
//     evaluating strategies, segments, and constraints.
//   - It's recommended to provide a context and set at least one of userId or sessionId
//     to get stable rollout and stickiness behavior.
//
// Fallback behavior (when the feature cannot be resolved):
//   - If FallbackFunc is non-nil, it is called and its result is returned.
//   - Else if Fallback is non-nil, *Fallback is returned.
//   - Else the result defaults to false.
type FeatureOptions struct {
	// Ctx provides the base request context for evaluation. If nil, the client's static
	// context is used.
	Ctx context.Context

	// Fallback is returned when the feature cannot be resolved and FallbackFunc is nil.
	Fallback *bool

	// FallbackFunc is called when the feature cannot be resolved. If set, it takes
	// precedence over Fallback.
	FallbackFunc FallbackFunc

	// Resolver bypasses repository lookup for resolving the feature by name.
	Resolver FeatureResolver
}

// VariantOptions controls how GetVariant evaluates a feature's variant.
//
// Resolver:
//   - If Resolver is non-nil, it is used to resolve the feature by name instead of the repository.
//     This bypasses the normal resolution path and should no longer be necessary in v6.
//     It exists for backwards compatibility and may be removed in a future release.
//
// Context:
//   - If Ctx is non-nil, it is merged with the client's static context and used when
//     evaluating strategies, segments, and constraints.
//   - It's recommended to provide a context and set at least one of userId or sessionId
//     to get stable rollout and stickiness behavior.
//
// Fallback behavior (when no variant can be resolved):
//   - If VariantFallbackFunc is non-nil, it is called and its result is returned.
//   - Else if VariantFallback is non-nil, *VariantFallback is returned.
//   - Else a default disabled variant is returned.
type VariantOptions struct {
	Ctx                 context.Context
	VariantFallback     *api.Variant
	VariantFallbackFunc VariantFallbackFunc
	Resolver            FeatureResolver
}

// VariantFallbackFunc represents a function to be called if the variant is not found.
type VariantFallbackFunc func(feature string, ctx *context.Context) *api.Variant

type variantOption struct {
	variantFallback     *api.Variant
	variantFallbackFunc VariantFallbackFunc
	ctx                 *context.Context
	resolver            FeatureResolver
}

type repositoryOptions struct {
	appName         string
	instanceId      string
	projectName     string
	url             url.URL
	backupPath      string
	refreshInterval time.Duration
	storage         Storage
	httpClient      *http.Client
	headers         http.Header
	isStreaming     bool
}

type metricsOptions struct {
	appName         string
	instanceId      string
	connectionId    string
	url             url.URL
	strategies      []string
	metricsInterval time.Duration
	disableMetrics  bool
	httpClient      *http.Client
	headers         http.Header
}
