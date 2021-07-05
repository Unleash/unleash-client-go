package unleash

import (
	"fmt"

	"net/url"
	"strings"
	"time"

	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/Unleash/unleash-client-go/v3/context"
	"github.com/Unleash/unleash-client-go/v3/internal/constraints"
	s "github.com/Unleash/unleash-client-go/v3/internal/strategies"
	"github.com/Unleash/unleash-client-go/v3/strategy"
)

const (
	deprecatedSuffix = "/features"
	clientName       = "unleash-client-go"
	clientVersion    = "3.2.3"
)

var defaultStrategies = []strategy.Strategy{
	*s.NewDefaultStrategy(),
	*s.NewApplicationHostnameStrategy(),
	*s.NewGradualRolloutRandomStrategy(),
	*s.NewGradualRolloutSessionId(),
	*s.NewGradualRolloutUserId(),
	*s.NewRemoteAddressStrategy(),
	*s.NewUserWithIdStrategy(),
	*s.NewFlexibleRolloutStrategy(),
}

// Client is a structure representing an API client of an Unleash server.
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
	onReady            chan struct{}
	close              chan struct{}
	closed             chan struct{}
	count              chan metric
	sent               chan MetricsData
	registered         chan ClientData
	staticContext      *context.Context
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

// NewClient creates a new client instance with the given options.
func NewClient(options ...ConfigOption) (*Client, error) {

	errChannels := errorChannels{
		errors:   make(chan error, 3),
		warnings: make(chan error, 3),
	}

	uc := &Client{
		options: configOption{
			environment:     "default",
			refreshInterval: 15 * time.Second,
			metricsInterval: 60 * time.Second,
			disableMetrics:  false,
			backupPath:      getTmpDirPath(),
			strategies:      []strategy.Strategy{},
		},
		errorChannels: errChannels,
		onReady:       make(chan struct{}),
		ready:         make(chan bool, 1),
		count:         make(chan metric),
		sent:          make(chan MetricsData),
		registered:    make(chan ClientData, 1),
		close:         make(chan struct{}),
		closed:        make(chan struct{}),
	}

	for _, opt := range options {
		opt(&uc.options)
	}

	uc.staticContext = &context.Context{
		Environment: uc.options.environment,
		AppName:     uc.options.appName,
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

	uc.repository = newRepository(
		repositoryOptions{
			backupPath:      uc.options.backupPath,
			url:             *parsedUrl,
			appName:         uc.options.appName,
			projectName:     uc.options.projectName,
			instanceId:      uc.options.instanceId,
			refreshInterval: uc.options.refreshInterval,
			storage:         uc.options.storage,
			httpClient:      uc.options.httpClient,
			customHeaders:   uc.options.customHeaders,
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

	uc.metrics = newMetrics(
		metricsOptions{
			appName:         uc.options.appName,
			instanceId:      uc.options.instanceId,
			strategies:      strategyNames,
			metricsInterval: uc.options.metricsInterval,
			url:             *parsedUrl,
			httpClient:      uc.options.httpClient,
			customHeaders:   uc.options.customHeaders,
			disableMetrics:  uc.options.disableMetrics,
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
			close(uc.onReady)
			if uc.repositoryListener != nil {
				uc.repositoryListener.OnReady()
			}
		case m := <-uc.count:
			if uc.metricsListener != nil {
				uc.metricsListener.OnCount(m.Name, m.Enabled)
			}
		case md := <-uc.sent:
			if uc.metricsListener != nil {
				uc.metricsListener.OnSent(md)
			}
		case cd := <-uc.registered:
			if uc.metricsListener != nil {
				uc.metricsListener.OnRegistered(cd)
			}
		case <-uc.close:
			close(uc.closed)
			return
		}
	}
}

// IsEnabled queries whether the specified feature is enabled or not.
//
// It is safe to call this method from multiple goroutines concurrently.
func (uc *Client) IsEnabled(feature string, options ...FeatureOption) (enabled bool) {
	defer func() {
		uc.metrics.count(feature, enabled)
	}()

	return uc.isEnabled(feature, options...)
}

// isEnabled abstracts away the details of checking if a toggle is turned on or off
// without metrics
func (uc *Client) isEnabled(feature string, options ...FeatureOption) (enabled bool) {
	f := uc.repository.getToggle(feature)

	var opts featureOption
	for _, o := range options {
		o(&opts)
	}

	ctx := uc.staticContext
	if opts.ctx != nil {
		ctx = ctx.Override(*opts.ctx)
	}

	if f == nil {
		if opts.fallbackFunc != nil {
			return opts.fallbackFunc(feature, ctx)
		} else if opts.fallback != nil {
			return *opts.fallback
		}
		return false
	}

	if !f.Enabled {
		return false
	}

	if len(f.Strategies) == 0 {
		return f.Enabled
	}

	for _, s := range f.Strategies {
		foundStrategy := uc.getStrategy(s.Name)
		if foundStrategy == nil {
			// TODO: warnOnce missingStrategy
			continue
		}

		if constraints.Check(ctx, s.Constraints) && foundStrategy.IsEnabled(s.Parameters, ctx) {
			return true
		}
	}
	return false
}

// GetVariant queries a variant as the specified feature is enabled.
//
// It is safe to call this method from multiple goroutines concurrently.
func (uc *Client) GetVariant(feature string, options ...VariantOption) *api.Variant {
	defaultVariant := api.GetDefaultVariant()

	if !uc.isEnabled(feature) {
		return defaultVariant;
	}

	f := uc.repository.getToggle(feature)

	var opts variantOption
	for _, o := range options {
		o(&opts)
	}

	ctx := uc.staticContext
	if (opts.ctx != nil) {
		ctx = ctx.Override(*opts.ctx)
	}

	if f == nil {
		if opts.variantFallbackFunc != nil {
			return opts.variantFallbackFunc(feature, ctx)
		} else if opts.variantFallback != nil {
			return opts.variantFallback
		}
		return defaultVariant
	}

	if !f.Enabled {
		return defaultVariant
	}

	if len(f.Variants) == 0 {
		return defaultVariant
	}

	variant := f.GetVariant(ctx)

	defer func() {
		uc.metrics.countVariants(feature, variant.Name)
	}()
	return variant
}

// Close stops the client from syncing data from the server.
func (uc *Client) Close() error {
	uc.repository.Close()
	uc.metrics.Close()
	if uc.options.listener != nil {
		// Wait for sync to exit.
		close(uc.close)
		<-uc.closed
	}
	return nil
}

// Errors returns the error channel for the client.
func (uc *Client) Errors() <-chan error {
	return uc.errors
}

// Warnings returns the warnings channel for the client.
func (uc *Client) Warnings() <-chan error {
	return uc.warnings
}

// Ready returns the ready channel for the client. A value will be available on
// the channel when the feature toggles have been loaded from the Unleash
// server.
func (uc *Client) Ready() <-chan bool {
	return uc.ready
}

// Count returns the count channel which gives an update when a toggle has been queried.
func (uc *Client) Count() <-chan metric {
	return uc.count
}

// Registered returns the registered signal indicating that the client has successfully connected to the
// metrics service.
func (uc *Client) Registered() <-chan ClientData {
	return uc.registered
}

// Sent returns the sent channel which receives data whenever the client has successfully sent metrics to
// the metrics service.
func (uc *Client) Sent() <-chan MetricsData {
	return uc.sent
}

func (uc *Client) getStrategy(name string) strategy.Strategy {
	for _, strategy := range uc.strategies {
		if strategy.Name() == name {
			return strategy
		}
	}
	return nil
}

// WaitForReady will block until the client has loaded the feature toggles from
// the Unleash server. It will return immediately if the toggles have already
// been loaded,
//
// It is safe to call this method from multiple goroutines concurrently.
func (uc *Client) WaitForReady() {
	<-uc.onReady
}

// ListFeatures returns all available features toggles.
func (uc *Client) ListFeatures() []api.Feature {
	return uc.repository.list()
}
