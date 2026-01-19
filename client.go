package unleash

import (
	"fmt"

	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Unleash/unleash-go-sdk/v6/api"
	"github.com/Unleash/unleash-go-sdk/v6/context"
	s "github.com/Unleash/unleash-go-sdk/v6/internal/strategies"
	"github.com/Unleash/unleash-go-sdk/v6/strategy"
)

const (
	deprecatedSuffix = "/features"
	clientName       = "unleash-go-sdk"
	clientVersion    = "5.0.3"
	specVersion      = "4.3.1"
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

// disabledVariantFeatureEnabled is similar to api.DISABLED_VARIANT but we want
// to discourage public usage so it's internal until there's a need to expose it.
var disabledVariantFeatureEnabled = &api.Variant{
	Name:           "disabled",
	Enabled:        false,
	FeatureEnabled: true,
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
	impressionListener ImpressionListener
	ready              chan bool
	onReady            chan struct{}
	update             chan bool
	close              chan struct{}
	closed             chan struct{}
	count              chan metric
	sent               chan MetricsData
	registered         chan ClientData
	impression         chan ImpressionEvent
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
	ready  chan bool
	update chan bool
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
		update:        make(chan bool, 1),
		count:         make(chan metric),
		sent:          make(chan MetricsData),
		registered:    make(chan ClientData, 1),
		impression:    make(chan ImpressionEvent, 10),
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

	if uc.options.listener == nil {
		uc.options.listener = &NoopListener{}
	}

	if eListener, ok := uc.options.listener.(ErrorListener); ok {
		uc.errorListener = eListener
	}
	if rListener, ok := uc.options.listener.(RepositoryListener); ok {
		uc.repositoryListener = rListener
	}
	if mListener, ok := uc.options.listener.(MetricListener); ok {
		uc.metricsListener = mListener
	}
	if iListener, ok := uc.options.listener.(ImpressionListener); ok {
		uc.impressionListener = iListener
	}

	defer func() {
		go uc.sync()
	}()

	if uc.options.url == "" {
		return nil, fmt.Errorf("unleash server URL missing")
	}

	if strings.HasSuffix(uc.options.url, deprecatedSuffix) {
		uc.warn(fmt.Errorf("unleash server URL %s should no longer link directly to /features", uc.options.url))
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
		return nil, fmt.Errorf("unleash client appName missing")
	}

	if uc.options.instanceId == "" {
		uc.options.instanceId = generateInstanceId()
	}

	connectionId := getConnectionId()

	headers := make(http.Header)
	if uc.options.customHeaders != nil {
		headers = uc.options.customHeaders
	}
	headers.Set("unleash-appname", uc.options.appName)
	headers.Set("unleash-sdk", fmt.Sprintf("%s:%s", clientName, clientVersion))
	headers.Set("unleash-connection-id", connectionId)

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
			headers:         headers,
			isStreaming:     uc.options.IsStreamingMode(),
		},
		repositoryChannels{
			errorChannels: errChannels,
			ready:         uc.ready,
			update:        uc.update,
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
			connectionId:    connectionId,
			strategies:      strategyNames,
			metricsInterval: uc.options.metricsInterval,
			url:             *parsedUrl,
			httpClient:      uc.options.httpClient,
			headers:         headers,
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
		case <-uc.update:
			if uc.repositoryListener != nil {
				uc.repositoryListener.OnUpdate()
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
		case ie := <-uc.impression:
			if uc.impressionListener != nil {
				uc.impressionListener.OnImpression(ie)
			}
		case <-uc.close:
			close(uc.update)
			close(uc.closed)
			return
		}
	}
}

// IsEnabled queries whether the specified feature is enabled or not.
//
// It is safe to call this method from multiple goroutines concurrently.
func (uc *Client) IsEnabled(feature string, options FeatureOptions) (enabled bool) {
	snapshot := uc.repository.snapshot()
	result, f := uc.isEnabled(feature, snapshot, options)
	enabled = result.Enabled

	defer func() {
		uc.metrics.count(feature, enabled)

		if f != nil && f.ImpressionData && uc.impressionListener != nil {
			ctx := uc.staticContext
			ctx = ctx.Override(options.Ctx)

			uc.impression <- ImpressionEvent{
				FeatureName: feature,
				EventType:   ImpressionEventTypeIsEnabled,
				Enabled:     enabled,
				Context:     ctx,
			}
		}
	}()

	return
}

// isEnabled abstracts away the details of checking if a toggle is turned on or off
// without metrics
func (uc *Client) isEnabled(
	feature string,
	snapshot *FeatureMemoryState,
	opts FeatureOptions,
) (api.StrategyResult, *api.Feature) {
	f := resolveToggle(snapshot, opts.Resolver, feature)

	ctx := uc.staticContext
	ctx = ctx.Override(opts.Ctx)

	if f == nil {
		return handleFallback(opts.FallbackFunc, opts.Fallback, feature, ctx), nil
	}

	result, err := snapshot.evaluateFeature(f, ctx, uc.strategies)
	if err != nil {
		uc.errors <- err
		return api.StrategyResult{Enabled: false}, f
	}

	return result, f
}

// GetVariant queries a variant as the specified feature is enabled.
//
// It is safe to call this method from multiple goroutines concurrently.
func (uc *Client) GetVariant(feature string, options VariantOptions) (variant *api.Variant) {
	snapshot := uc.repository.snapshot()
	variant = uc.getVariant(feature, snapshot, options)

	defer func() {
		uc.metrics.countVariants(feature, variant.FeatureEnabled, variant.Name)

		f := snapshot.Features[feature]
		if f != nil && f.ImpressionData && uc.impressionListener != nil {
			ctx := uc.staticContext
			ctx = ctx.Override(options.Ctx)

			uc.impression <- ImpressionEvent{
				FeatureName: feature,
				EventType:   ImpressionEventTypeGetVariant,
				Enabled:     variant.FeatureEnabled,
				Variant:     variant.Name,
				Context:     ctx,
			}
		}
	}()
	return
}

func (uc *Client) getVariant(feature string, snapshot *FeatureMemoryState, opts VariantOptions) *api.Variant {
	internal := variantOption{}

	internal.variantFallback = opts.VariantFallback
	internal.variantFallbackFunc = opts.VariantFallbackFunc
	internal.resolver = opts.Resolver
	internal.ctx = &opts.Ctx

	ctx := uc.staticContext
	ctx = ctx.Override(opts.Ctx)

	var strategyResult api.StrategyResult
	var f *api.Feature
	strategyResult, f = uc.isEnabled(feature, snapshot, FeatureOptions{
		Ctx:      *ctx,
		Resolver: internal.resolver,
	})

	getFallbackVariant := func(featureEnabled bool) *api.Variant {
		if internal.variantFallbackFunc != nil {
			return internal.variantFallbackFunc(feature, ctx)
		} else if internal.variantFallback != nil {
			return internal.variantFallback
		}

		if featureEnabled {
			return disabledVariantFeatureEnabled
		}
		return api.GetDefaultVariant()
	}

	if !strategyResult.Enabled {
		return getFallbackVariant(false)
	}

	if f == nil || !f.Enabled {
		return getFallbackVariant(false)
	}

	if strategyResult.Variant != nil {
		return strategyResult.Variant
	}

	if len(f.Variants) == 0 {
		return getFallbackVariant(true)
	}

	return api.VariantCollection{
		GroupId:  f.Name,
		Variants: f.Variants,
	}.GetVariant(ctx, nil)
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

// Impression returns the impression channel which gives an update when a feature with impression data has been evaluated.
func (uc *Client) Impression() <-chan ImpressionEvent {
	return uc.impression
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

func handleFallback(fallbackFunc FallbackFunc, fallback *bool, featureName string, ctx *context.Context) api.StrategyResult {
	if fallbackFunc != nil {
		return api.StrategyResult{
			Enabled: fallbackFunc(featureName, ctx),
		}
	} else if fallback != nil {
		return api.StrategyResult{
			Enabled: *fallback,
		}
	}

	return api.StrategyResult{
		Enabled: false,
	}
}

func resolveToggle(snapshot *FeatureMemoryState, resolver FeatureResolver, featureName string) *api.Feature {
	if resolver != nil {
		return resolver(featureName)
	} else {
		return snapshot.Features[featureName]
	}
}
