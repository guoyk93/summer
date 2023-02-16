package summer

type options struct {
	concurrency      int
	readinessCascade int64
	readinessPath    string
	livenessPath     string
	metricsPath      string
}

// Option a function configuring [App]
type Option func(opts *options)

// WithConcurrency set maximum concurrent requests of [App].
//
// A value <= 0 means unlimited
func WithConcurrency(c int) Option {
	return func(opts *options) {
		opts.concurrency = c
	}
}

// WithReadinessCascade set maximum continuous failed Readiness Checks after which Liveness CheckFunc start to fail.
//
// Failing Liveness Checks could trigger a Pod restart.
//
// A value <= 0 means disabled
func WithReadinessCascade(rc int) Option {
	return func(opts *options) {
		opts.readinessCascade = int64(rc)
	}
}

// WithReadinessPath set readiness check path
func WithReadinessPath(s string) Option {
	return func(opts *options) {
		opts.readinessPath = s
	}
}

// WithLivenessPath set liveness path
func WithLivenessPath(s string) Option {
	return func(opts *options) {
		opts.livenessPath = s
	}
}

// WithMetricsPath set metrics path
func WithMetricsPath(s string) Option {
	return func(opts *options) {
		opts.metricsPath = s
	}
}
