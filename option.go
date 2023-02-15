package summer

// Option a function configuring [App]
type Option func(a *app)

// WithConcurrency set maximum concurrent requests of [App].
//
// A value <= 0 means unlimited
func WithConcurrency(c int) Option {
	return func(a *app) {
		a.optConcurrency = c
	}
}

// WithReadinessCascade set maximum continuous failed Readiness Checks after which Liveness Check start to fail.
//
// Failing Liveness Checks could trigger a Pod restart.
//
// A value <= 0 means disabled
func WithReadinessCascade(rc int) Option {
	return func(a *app) {
		a.optReadinessCascade = int64(rc)
	}
}
