package feature

import "github.com/Unleash/unleash-client-go/v3/context"

// Options is a struct representing the various options supported when querying if a
// feature is enabled or not.
type Options struct {
	fallback *bool
	ctx      *context.Context
}

// Option represents a single option
type Option func(*Options)

// WithFallback specifies what the value should be if the feature toggle is not found on the
// unleash service.
func WithFallback(fallback bool) Option {
	return func(opts *Options) {
		opts.fallback = &fallback
	}
}

// WithContext allows the user to provide a context that will be passed into the active strategy
// for determining if a specified feature should be enabled or not.
func WithContext(ctx context.Context) Option {
	return func(opts *Options) {
		opts.ctx = &ctx
	}
}

// FlattenOptions takes a list of feature options and flattens them into a struct that can be
// passed around.
func FlattenOptions(options ...Option) Options {
	var opts Options
	for _, o := range options {
		o(&opts)
	}
	return opts
}

// Fallback returns the fallback value of the feature or nil if one has not
// been specified.
func (opts Options) Fallback() *bool {
	return opts.fallback
}

// Context returns the context for the feature or nil if one has not been specified.
func (opts Options) Context() *context.Context {
	return opts.ctx
}
