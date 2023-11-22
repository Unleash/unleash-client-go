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
	clientVersion    = "3.9.1"
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
	defer func() {
		go uc.sync()
	}()

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

	return uc.isEnabled(feature, options...).Enabled
}

// isEnabled abstracts away the details of checking if a toggle is turned on or off
// without metrics
func (uc *Client) isEnabled(feature string, options ...FeatureOption) api.StrategyResult {
	var opts featureOption
	for _, o := range options {
		o(&opts)
	}

	f := resolveToggle(uc, opts, feature)

	ctx := uc.staticContext
	if opts.ctx != nil {
		ctx = ctx.Override(*opts.ctx)
	}

	if f == nil {
		return handleFallback(opts, feature, ctx)
	}

	if f.Dependencies != nil && len(*f.Dependencies) > 0 {
		dependenciesSatisfied := uc.isParentDependencySatisfied(f, *ctx)

		if !dependenciesSatisfied {
			return api.StrategyResult{
				Enabled: false,
			}
		}
	}

	if !f.Enabled {
		return api.StrategyResult{
			Enabled: false,
		}
	}

	if len(f.Strategies) == 0 {
		return api.StrategyResult{
			Enabled: f.Enabled,
		}
	}

	for _, s := range f.Strategies {
		foundStrategy := uc.getStrategy(s.Name)
		if foundStrategy == nil {
			// TODO: warnOnce missingStrategy
			continue
		}

		segmentConstraints, err := uc.repository.resolveSegmentConstraints(s)

		if err != nil {
			uc.errors <- err
			return api.StrategyResult{
				Enabled: false,
			}
		}

		allConstraints := make([]api.Constraint, 0)
		allConstraints = append(allConstraints, segmentConstraints...)
		allConstraints = append(allConstraints, s.Constraints...)

		if ok, err := constraints.Check(ctx, allConstraints); err != nil {
			uc.errors <- err
		} else if ok && foundStrategy.IsEnabled(s.Parameters, ctx) {
			if s.Variants != nil && len(s.Variants) > 0 {
				groupIdValue := s.Parameters[strategy.ParamGroupId]
				groupId, ok := groupIdValue.(string)
				if !ok {
					return api.StrategyResult{
						Enabled: false,
					}
				}

				return api.StrategyResult{
					Enabled: true,
					Variant: api.VariantCollection{
						GroupId:  groupId,
						Variants: s.Variants,
					}.GetVariant(ctx),
				}
			} else {
				return api.StrategyResult{
					Enabled: true,
				}
			}
		}
	}

	return api.StrategyResult{
		Enabled: false,
	}
}

func (uc *Client) isParentDependencySatisfied(feature *api.Feature, context context.Context) bool {
	warnOnce := &WarnOnce{}

	dependenciesSatisfied := func(parent api.Dependency) bool {
		parentToggle := uc.repository.getToggle(parent.Feature)

		if parentToggle == nil {
			warnOnce.Warn("the parent toggle was not found in the cache, the evaluation of this dependency will always be false")
			return false
		}

		if parentToggle.Dependencies != nil && len(*parentToggle.Dependencies) > 0 {
			return false
		}

		enabledResult := uc.isEnabled(parent.Feature, WithContext(context))
		// According to the schema, if the enabled property is absent we assume it's true.
		if parent.Enabled == nil || *parent.Enabled == true {
			if parent.Variants != nil && len(*parent.Variants) > 0 && enabledResult.Variant != nil {
				return enabledResult.Enabled && contains(*parent.Variants, enabledResult.Variant.Name)
			}
			return enabledResult.Enabled
		}

		return !enabledResult.Enabled
	}

	allDependenciesSatisfied := every(*feature.Dependencies, func(parent interface{}) bool {
		return dependenciesSatisfied(parent.(api.Dependency))
	})

	if !allDependenciesSatisfied {
		return false
	}

	return true
}

// GetVariant queries a variant as the specified feature is enabled.
//
// It is safe to call this method from multiple goroutines concurrently.
func (uc *Client) GetVariant(feature string, options ...VariantOption) *api.Variant {
	variant := uc.getVariantWithoutMetrics(feature, options...)
	defer func() {
		uc.metrics.countVariants(feature, variant.Enabled, variant.Name)
	}()
	return variant
}

// getVariantWithoutMetrics abstracts away the logic for resolving a variant without metrics
func (uc *Client) getVariantWithoutMetrics(feature string, options ...VariantOption) *api.Variant {
	defaultVariant := api.GetDefaultVariant()
	var opts variantOption
	for _, o := range options {
		o(&opts)
	}

	ctx := uc.staticContext
	if opts.ctx != nil {
		ctx = ctx.Override(*opts.ctx)
	}

	var strategyResult api.StrategyResult

	if opts.resolver != nil {
		strategyResult = uc.isEnabled(feature, WithContext(*ctx), WithResolver(opts.resolver))
	} else {
		strategyResult = uc.isEnabled(feature, WithContext(*ctx))
	}

	if !strategyResult.Enabled {
		return defaultVariant
	}

	var f *api.Feature
	if opts.resolver != nil {
		f = opts.resolver(feature)
	} else {
		f = uc.repository.getToggle(feature)
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

	if strategyResult.Variant != nil {
		return strategyResult.Variant
	}

	if len(f.Variants) == 0 {
		return defaultVariant
	}

	return api.VariantCollection{
		GroupId:  f.Name,
		Variants: f.Variants,
	}.GetVariant(ctx)
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

func resolveToggle(unleashClient *Client, opts featureOption, featureName string) *api.Feature {
	var feature *api.Feature
	if opts.resolver != nil {
		feature = opts.resolver(featureName)
	} else {
		feature = unleashClient.repository.getToggle(featureName)
	}

	return feature
}

func handleFallback(opts featureOption, featureName string, ctx *context.Context) api.StrategyResult {
	if opts.fallbackFunc != nil {
		return api.StrategyResult{
			Enabled: opts.fallbackFunc(featureName, ctx),
		}
	} else if opts.fallback != nil {
		return api.StrategyResult{
			Enabled: *opts.fallback,
		}
	}

	return api.StrategyResult{
		Enabled: false,
	}
}
