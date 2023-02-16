package summer

import (
	"context"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync/atomic"
)

// CheckerFunc health check function, see [App.Check]
type CheckerFunc func(ctx context.Context) (err error)

// HandlerFunc handler func with [Context] as argument
type HandlerFunc[T Context] func(ctx T)

// App the main interface of [summer]
type App[T Context] interface {
	http.Handler

	// CheckFunc register a checker function with given name
	//
	// Invoking '/debug/ready' will evaluate all registered checker functions
	CheckFunc(name string, fn CheckerFunc)

	// HandleFunc register an action function with given path pattern
	//
	// This function is similar with [http.ServeMux.HandleFunc]
	HandleFunc(pattern string, fn HandlerFunc[T])
}

type app[T Context] struct {
	contextFactory ContextFactory[T]

	opts options

	checkers map[string]CheckerFunc

	mux *http.ServeMux

	h  http.Handler
	ph http.Handler

	pprof http.Handler

	cc chan struct{}

	readinessFailed int64
}

func (a *app[T]) CheckFunc(name string, fn CheckerFunc) {
	a.checkers[name] = fn
}

func (a *app[T]) executeCheckers(ctx context.Context) (r string, failed bool) {
	sb := &strings.Builder{}
	for k, fn := range a.checkers {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(k)
		sb.WriteString(": ")
		if err := fn(ctx); err == nil {
			sb.WriteString("OK")
		} else {
			failed = true
			sb.WriteString(err.Error())
		}
	}
	r = sb.String()
	if r == "" {
		r = "OK"
	}
	return
}

func (a *app[T]) HandleFunc(pattern string, fn HandlerFunc[T]) {
	a.mux.Handle(
		pattern,
		otelhttp.WithRouteTag(
			pattern,
			http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				c := a.contextFactory(rw, req)
				func() {
					defer c.Perform()
					fn(c)
				}()
			}),
		),
	)
}

func (a *app[T]) initialize() {
	// checkers
	a.checkers = map[string]CheckerFunc{}

	// promhttp handler
	a.ph = promhttp.Handler()

	// pprof handler
	{
		m := &http.ServeMux{}
		m.HandleFunc("/debug/pprof/", pprof.Index)
		m.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		m.HandleFunc("/debug/pprof/profile", pprof.Profile)
		m.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		m.HandleFunc("/debug/pprof/trace", pprof.Trace)
		a.pprof = m
	}

	// handler
	a.mux = &http.ServeMux{}
	a.h = otelhttp.NewHandler(a.mux, "http")

	// concurrency control
	if a.opts.concurrency > 0 {
		a.cc = make(chan struct{}, a.opts.concurrency)
		for i := 0; i < a.opts.concurrency; i++ {
			a.cc <- struct{}{}
		}
	}
}

func (a *app[T]) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// alive, ready, metrics
	if req.URL.Path == a.opts.readinessPath {
		// readiness first, works when readinessPath == livenessPath
		r, failed := a.executeCheckers(req.Context())
		status := http.StatusOK
		if failed {
			atomic.AddInt64(&a.readinessFailed, 1)
			status = http.StatusInternalServerError
		} else {
			atomic.StoreInt64(&a.readinessFailed, 0)
		}
		respondInternal(rw, r, status)
		return
	} else if req.URL.Path == a.opts.livenessPath {
		if a.opts.readinessCascade > 0 && atomic.LoadInt64(&a.readinessFailed) > a.opts.readinessCascade {
			respondInternal(rw, "CASCADED", http.StatusInternalServerError)
		} else {
			respondInternal(rw, "OK", http.StatusOK)
		}
		return
	} else if req.URL.Path == a.opts.metricsPath {
		a.ph.ServeHTTP(rw, req)
		return
	}

	// pprof
	if strings.HasPrefix(req.URL.Path, "/debug/") {
		a.pprof.ServeHTTP(rw, req)
		return
	}

	// concurrency control
	if a.cc != nil {
		<-a.cc
		defer func() {
			a.cc <- struct{}{}
		}()
	}

	a.h.ServeHTTP(rw, req)
}

// New create an [App] with a custom [ContextFactory] and additional [Option]
func New[T Context](cf ContextFactory[T], opts ...Option) App[T] {
	a := &app[T]{
		contextFactory: cf,

		opts: options{
			concurrency:      128,
			readinessCascade: 5,
			readinessPath:    DefaultReadinessPath,
			livenessPath:     DefaultLivenessPath,
			metricsPath:      DefaultMetricsPath,
		},
	}
	for _, opt := range opts {
		opt(&a.opts)
	}
	a.initialize()
	return a
}

// BasicApp basic app is an [App] using vanilla [Context]
type BasicApp = App[Context]

// Basic create an [App] with vanilla [Context] and additional [Option]
func Basic(opts ...Option) BasicApp {
	return New(BasicContext, opts...)
}
