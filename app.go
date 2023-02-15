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

// Checker a general health checker func, check App#Check()
type Checker func(ctx context.Context) (err error)

// Options options to create a [App]
type Options struct {
	// Concurrency maximum concurrent request in-flight
	Concurrency int
	// CascadeReadiness maximum count of continuous failed readiness checks, after that, liveness probe will fail, triggering pod restart
	CascadeReadiness int
}

func (opts Options) sanitize() Options {
	if opts.Concurrency <= 0 {
		opts.Concurrency = 128
	}
	if opts.CascadeReadiness <= 0 {
		opts.CascadeReadiness = 5
	}
	return opts
}

// App the main interface of [summer]
type App interface {
	http.Handler

	// Check register a service checker with given name
	Check(name string, fn Checker)

	// Handle register an action with given path pattern
	Handle(pattern string, fn func(c Context))
}

type app struct {
	opts Options

	checkers map[string]Checker

	mux *http.ServeMux

	h     http.Handler
	debug http.Handler

	cc chan struct{}

	readinessFailed int64
}

func (a *app) Check(name string, fn Checker) {
	a.checkers[name] = fn
}

func (a *app) executeCheckers(ctx context.Context) (r string, failed bool) {
	sb := &strings.Builder{}
	for k, fn := range a.checkers {
		sb.WriteString(k)
		sb.WriteString(": ")
		if err := fn(ctx); err == nil {
			sb.WriteString("OK")
		} else {
			failed = true
			sb.WriteString(err.Error())
		}
		sb.WriteString("\n")
	}
	r = sb.String()
	if r == "" {
		r = "OK"
	}
	return
}

func (a *app) Handle(pattern string, fn func(c Context)) {
	a.mux.Handle(
		pattern,
		otelhttp.WithRouteTag(
			pattern,
			http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				c := newContext(rw, req)
				func() {
					defer c.Perform()
					fn(c)
				}()
			}),
		),
	)
}

func (a *app) initialize() {
	// checkers
	a.checkers = map[string]Checker{}

	// debug handler
	m := &http.ServeMux{}
	m.HandleFunc(DebugPathAlive, func(rw http.ResponseWriter, req *http.Request) {
		if a.readinessFailed > int64(a.opts.CascadeReadiness) {
			http.Error(rw, "CASCADED", http.StatusInternalServerError)
		} else {
			http.Error(rw, "OK", http.StatusOK)
		}
	})
	m.HandleFunc(DebugPathReady, func(rw http.ResponseWriter, req *http.Request) {
		r, failed := a.executeCheckers(req.Context())
		status := http.StatusOK
		if failed {
			atomic.AddInt64(&a.readinessFailed, 1)
			status = http.StatusInternalServerError
		} else {
			atomic.StoreInt64(&a.readinessFailed, 0)
		}
		http.Error(rw, r, status)
	})
	m.Handle(DebugPathMetrics, promhttp.Handler())
	m.HandleFunc("/debug/pprof/", pprof.Index)
	m.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	m.HandleFunc("/debug/pprof/profile", pprof.Profile)
	m.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	m.HandleFunc("/debug/pprof/trace", pprof.Trace)
	a.debug = m

	// handler
	a.mux = &http.ServeMux{}
	a.h = otelhttp.NewHandler(a.mux, "http")

	// concurrency control
	a.cc = make(chan struct{}, a.opts.Concurrency)
	for i := 0; i < a.opts.Concurrency; i++ {
		a.cc <- struct{}{}
	}
}

func (a *app) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, DebugPathPrefix) {
		a.debug.ServeHTTP(rw, req)
		return
	}

	// concurrency control
	<-a.cc
	defer func() {
		a.cc <- struct{}{}
	}()

	a.h.ServeHTTP(rw, req)
}

// New create a [App]
func New(opts Options) App {
	opts = opts.sanitize()
	a := &app{opts: opts}
	a.initialize()
	return a
}
