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
type HandlerFunc func(ctx Context)

// App the main interface of [summer]
type App interface {
	http.Handler

	// CheckFunc register a checker function with given name
	//
	// Invoking '/debug/ready' will evaluate all registered checker functions
	CheckFunc(name string, fn CheckerFunc)

	// HandleFunc register an action function with given path pattern
	//
	// This function is similar with [http.ServeMux.HandleFunc]
	HandleFunc(pattern string, fn HandlerFunc)
}

type app struct {
	optConcurrency      int
	optReadinessCascade int64

	checkers map[string]CheckerFunc

	mux *http.ServeMux

	h     http.Handler
	debug http.Handler

	cc chan struct{}

	readinessFailed int64
}

func (a *app) CheckFunc(name string, fn CheckerFunc) {
	a.checkers[name] = fn
}

func (a *app) executeCheckers(ctx context.Context) (r string, failed bool) {
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

func (a *app) HandleFunc(pattern string, fn HandlerFunc) {
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
	a.checkers = map[string]CheckerFunc{}

	// debug handler
	m := &http.ServeMux{}
	m.HandleFunc(DebugPathAlive, func(rw http.ResponseWriter, req *http.Request) {
		if a.optReadinessCascade > 0 && atomic.LoadInt64(&a.readinessFailed) > a.optReadinessCascade {
			respondText(rw, "CASCADED", http.StatusInternalServerError)
		} else {
			respondText(rw, "OK", http.StatusOK)
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
		respondText(rw, r, status)
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
	if a.optConcurrency > 0 {
		a.cc = make(chan struct{}, a.optConcurrency)
		for i := 0; i < a.optConcurrency; i++ {
			a.cc <- struct{}{}
		}
	}
}

func (a *app) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, DebugPathPrefix) {
		a.debug.ServeHTTP(rw, req)
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

// New create an [App] with optional [Option]
func New(opts ...Option) App {
	a := &app{
		optReadinessCascade: 5,
		optConcurrency:      128,
	}
	for _, opt := range opts {
		opt(a)
	}
	a.initialize()
	return a
}
