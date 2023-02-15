package summer

type options struct {
	concurrency      int
	readinessCascade int64
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
