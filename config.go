package unleash_client_go

import (
	"github.com/unleash/unleash-client-go/strategy"
	"time"
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
}

type ConfigOption func(*configOption)

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
