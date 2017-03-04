package unleash

import (
	"net/http"
	"net/url"
	"time"

	"github.com/unleash/unleash-client-go/context"
	"github.com/unleash/unleash-client-go/strategy"
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
}

type ConfigOption func(*configOption)

func WithListener(listener interface{}) ConfigOption {
	return func(o *configOption) {
		o.listener = listener
	}
}

func WithAppName(appName string) ConfigOption {
	return func(o *configOption) {
		o.appName = appName
	}
}

func WithInstanceId(instanceId string) ConfigOption {
	return func(o *configOption) {
		o.instanceId = instanceId
	}
}

func WithUrl(url string) ConfigOption {
	return func(o *configOption) {
		o.url = url
	}
}

func WithRefreshInterval(refreshInterval time.Duration) ConfigOption {
	return func(o *configOption) {
		o.refreshInterval = refreshInterval
	}
}

func WithMetricsInterval(metricsInterval time.Duration) ConfigOption {
	return func(o *configOption) {
		o.metricsInterval = metricsInterval
	}
}

func WithDisableMetrics(disableMetrics bool) ConfigOption {
	return func(o *configOption) {
		o.disableMetrics = disableMetrics
	}
}

func WithBackupPath(backupPath string) ConfigOption {
	return func(o *configOption) {
		o.backupPath = backupPath
	}
}

func WithStrategies(strategies ...strategy.Strategy) ConfigOption {
	return func(o *configOption) {
		o.strategies = strategies
	}
}

type featureOption struct {
	fallback *bool
	ctx      *context.Context
}

type FeatureOption func(*featureOption)

func WithFallback(fallback bool) FeatureOption {
	return func(opts *featureOption) {
		opts.fallback = &fallback
	}
}

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
}

type metricsOptions struct {
	appName         string
	instanceID      string
	url             url.URL
	strategies      []string
	metricsInterval time.Duration
	disableMetrics  bool
	httpClient      *http.Client
}
