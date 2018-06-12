package unleash

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Unleash/unleash-client-go/api"
	s "github.com/Unleash/unleash-client-go/internal/strategies"
	"github.com/Unleash/unleash-client-go/strategy"
)

const (
	deprecatedSuffix = "/features"
	clientName       = "unleash-client-go"
	clientVersion    = "2.0.0"
)

var defaultStrategies = []strategy.Strategy{
	*s.NewDefaultStrategy(),
	*s.NewApplicationHostnameStrategy(),
	*s.NewGradualRolloutRandomStrategy(),
	*s.NewGradualRolloutSessionId(),
	*s.NewGradualRolloutUserId(),
	*s.NewRemoteAddressStrategy(),
	*s.NewUserWithIdStrategy(),
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
	closed             chan bool
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

// NewClient creates a new client instance with the given options.
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
		closed:        make(chan bool, 1),
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

	parsedURL, err := url.Parse(uc.options.url)
	if err != nil {
		return nil, err
	}

	if uc.options.appName == "" {
		return nil, fmt.Errorf("Unleash client appName missing")
	}

	if uc.options.instanceId == "" {
		uc.options.instanceId = generateInstanceID()
	}

	uc.repository = newRepository(
		repositoryOptions{
			backupPath:      uc.options.backupPath,
			url:             *parsedURL,
			appName:         uc.options.appName,
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
			url:             *parsedURL,
			httpClient:      uc.options.httpClient,
			customHeaders:   uc.options.customHeaders,
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
		case <-uc.closed:
			return
		}
	}
}

// IsEnabled queries whether or not the specified feature is enabled or not.
func (uc Client) IsEnabled(feature string, options ...FeatureOption) (enabled bool) {
	defer func() {
		uc.metrics.count(feature, enabled)
	}()

	f := uc.repository.getToggle(feature)

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
		if foundStrategy.IsEnabled(s.Parameters, opts.ctx) {
			return true
		}
	}
	return false
}

// GetFeaturesByPattern retrieves all features whose ID match the given pattern
func (uc Client) GetFeaturesByPattern(pattern string) []api.Feature {
	result := make([]api.Feature, 0)
	r, err := regexp.Compile(pattern)
	if err != nil {
		return result
	}
	features := uc.repository.getAllToggles()
	for _, f := range features {
		if r.Match([]byte(f.Name)) {
			ft := uc.GetFeature(f.Name)
			if ft != nil {
				result = append(result, *ft)
			}
		}
	}
	return result
}

// GetFeature queries the feature with the given name.
func (uc Client) GetFeature(name string) *api.Feature {
	f := uc.repository.getToggle(name)
	if f == nil {
		return nil
	}
	strategies := make([]api.Strategy, len(f.Strategies))
	for i, s := range f.Strategies {
		strategies[i] = api.Strategy{
			Name:       s.Name,
			Parameters: s.Parameters,
		}
	}
	return &api.Feature{
		Name:        f.Name,
		Description: f.Description,
		Enabled:     f.Enabled,
		Strategies:  strategies,
	}

}

// GetFeaturesByStrategy retrieves all features whose used a given Strategy
func (uc Client) GetFeaturesByStrategy(strategyName string) []api.Feature {
	result := make([]api.Feature, 0)
	features := uc.repository.getAllToggles()
	for _, feat := range features {
		for _, str := range feat.Strategies {
			if str.Name == strategyName {
				ft := uc.GetFeature(feat.Name)
				result = append(result, *ft)
			}
		}
	}
	return result
}

// Close stops the client from syncing data from the server.
func (uc *Client) Close() error {
	uc.repository.Close()
	uc.metrics.Close()
	uc.closed <- true
	return nil
}

// Errors returns the error channel for the client.
func (uc Client) Errors() <-chan error {
	return uc.errors
}

// Warnings returns the warnings channel for the client.
func (uc Client) Warnings() <-chan error {
	return uc.warnings
}

// Ready returns the ready signal for the client and is fired after the repository of locally stored
// toggles has been loaded.
func (uc Client) Ready() <-chan bool {
	return uc.ready
}

// Count returns the count channel which gives an update when a toggle has been queried.
func (uc Client) Count() <-chan metric {
	return uc.count
}

// Registered returns the registered signal indicating that the client has successfully connected to the
// metrics service.
func (uc Client) Registered() <-chan ClientData {
	return uc.registered
}

// Sent returns the sent channel which receives data whenever the client has successfully sent metrics to
// the metrics service.
func (uc Client) Sent() <-chan MetricsData {
	return uc.sent
}

func (uc Client) getStrategy(name string) strategy.Strategy {
	for _, strategy := range uc.strategies {
		if strategy.Name() == name {
			return strategy
		}
	}
	return nil
}
