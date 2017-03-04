package unleash

import (
	"fmt"
	s "github.com/unleash/unleash-client-go/internal/strategies"
	"github.com/unleash/unleash-client-go/strategy"
	"net/url"
	"strings"
	"time"
)

const deprecatedSuffix = "/features"

var defaultStrategies = []strategy.Strategy{
	*s.NewDefaultStrategy(),
	*s.NewApplicationHostnameStrategy(),
	*s.NewGradualRolloutRandomStrategy(),
	*s.NewGradualRolloutSessionId(),
	*s.NewGradualRolloutUserId(),
	*s.NewRemoteAddressStrategy(),
	*s.NewUserWithIdStrategy(),
}

type Client struct {
	errorChannels
	options            configOption
	repository         *repository
	metrics            *metrics
	strategies         []strategy.Strategy
	errorListener      ErrorListener
	metricsListener    MetricListener
	repositoryListener RepositoryListener
	ready              chan bool
	count              chan metric
	sent               chan MetricsData
	registered         chan ClientData
}

type errorChannels struct {
	errors   chan error
	warnings chan error
}

func (ec errorChannels) warn(err error) {
	ec.warnings <- err
}

func (ec errorChannels) err(err error) {
	ec.errors <- err
}

type repositoryChannels struct {
	errorChannels
	ready chan bool
}

type metricsChannels struct {
	errorChannels
	count      chan metric
	sent       chan MetricsData
	registered chan ClientData
}

func NewClient(options ...ConfigOption) (*Client, error) {

	errChannels := errorChannels{
		errors:   make(chan error, 3),
		warnings: make(chan error, 3),
	}

	uc := &Client{
		options: configOption{
			refreshInterval: 15 * time.Second,
			metricsInterval: 60 * time.Second,
			disableMetrics:  false,
			backupPath:      getTmpDirPath(),
			strategies:      []strategy.Strategy{},
		},
		errorChannels: errChannels,
		ready:         make(chan bool, 1),
		count:         make(chan metric),
		sent:          make(chan MetricsData),
		registered:    make(chan ClientData, 1),
	}

	for _, opt := range options {
		opt(&uc.options)
	}

	if uc.options.listener != nil {
		if eListener, ok := uc.options.listener.(ErrorListener); ok {
			uc.errorListener = eListener
		}
		if rListener, ok := uc.options.listener.(RepositoryListener); ok {
			uc.repositoryListener = rListener
		}
		if mListener, ok := uc.options.listener.(MetricListener); ok {
			uc.metricsListener = mListener
		}
		defer func() {
			go uc.sync()
		}()
	}

	if uc.options.url == "" {
		return nil, fmt.Errorf("Unleash server URL missing")
	}

	if strings.HasSuffix(uc.options.url, deprecatedSuffix) {
		uc.warn(fmt.Errorf("Unleash server URL %s should no longer link directly to /features", uc.options.url))
		uc.options.url = strings.TrimSuffix(uc.options.url, deprecatedSuffix)
	}

	if !strings.HasSuffix(uc.options.url, "/") {
		uc.options.url += "/"
	}

	parsedUrl, err := url.Parse(uc.options.url)
	if err != nil {
		return nil, err
	}

	if uc.options.appName == "" {
		return nil, fmt.Errorf("Unleash client appName missing")
	}

	if uc.options.instanceId == "" {
		uc.options.instanceId = generateInstanceId()
	}

	uc.repository = NewRepository(
		RepositoryOptions{
			BackupPath:      uc.options.backupPath,
			Url:             *parsedUrl,
			AppName:         uc.options.appName,
			InstanceId:      uc.options.instanceId,
			RefreshInterval: uc.options.refreshInterval,
		},
		repositoryChannels{
			errorChannels: errChannels,
			ready:         uc.ready,
		},
	)

	uc.strategies = append(defaultStrategies, uc.options.strategies...)

	strategyNames := make([]string, len(uc.strategies))
	for i, strategy := range uc.strategies {
		strategyNames[i] = strategy.Name()
	}

	uc.metrics = NewMetrics(
		MetricsOptions{
			AppName:         uc.options.appName,
			InstanceID:      uc.options.instanceId,
			Strategies:      strategyNames,
			MetricsInterval: uc.options.metricsInterval,
			BucketInterval:  uc.options.metricsInterval,
			Url:             *parsedUrl,
		},
		metricsChannels{
			errorChannels: errChannels,
			count:         uc.count,
			sent:          uc.sent,
			registered:    uc.registered,
		},
	)

	return uc, nil
}

func (uc *Client) sync() {
	for {
		select {
		case e := <-uc.errors:
			if uc.errorListener != nil {
				uc.errorListener.OnError(e)
			}
		case w := <-uc.warnings:
			if uc.errorListener != nil {
				uc.errorListener.OnWarning(w)
			}
		case <-uc.ready:
			if uc.repositoryListener != nil {
				uc.repositoryListener.OnReady()
			}
		case m := <-uc.count:
			if uc.metricsListener != nil {
				uc.metricsListener.OnCount(m.name, m.enabled)
			}
		case md := <-uc.sent:
			if uc.metricsListener != nil {
				uc.metricsListener.OnSent(md)
			}
		case cd := <-uc.registered:
			if uc.metricsListener != nil {
				uc.metricsListener.OnRegistered(cd)
			}
		}
	}
}

func (uc Client) IsEnabled(feature string, options ...FeatureOption) (enabled bool) {
	defer func() {
		uc.metrics.count(feature, enabled)
	}()

	f := uc.repository.GetToggle(feature)

	var opts featureOption
	for _, o := range options {
		o(&opts)
	}

	if f == nil {
		if opts.fallback != nil {
			return *opts.fallback
		}
		return false
	}

	if !f.Enabled {
		return false
	}

	for _, s := range f.Strategies {
		foundStrategy := uc.getStrategy(s.Name)
		if foundStrategy == nil {
			// TODO: warnOnce missingStrategy
			continue
		}
		return foundStrategy.IsEnabled(f.Parameters, opts.ctx)
	}
	return false
}

func (uc *Client) Close() error {
	uc.repository.Close()
	uc.metrics.Close()
	return nil
}

func (uc Client) getStrategy(name string) strategy.Strategy {
	for _, strategy := range uc.strategies {
		if strategy.Name() == name {
			return strategy
		}
	}
	return nil
}
