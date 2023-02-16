package summer

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync"
	"sync/atomic"
)

// LifecycleFunc lifecycle function for component
type LifecycleFunc func(ctx context.Context) (err error)

// HandlerFunc handler func with [Context] as argument
type HandlerFunc[T Context] func(ctx T)

// App the main interface of [summer]
type App[T Context] interface {
	http.Handler

	// Component register a component
	//
	// In order of `startup`, `check` and `shutdown`
	Component(name string, fns ...LifecycleFunc)

	// HandleFunc register an action function with given path pattern
	//
	// This function is similar with [http.ServeMux.HandleFunc]
	HandleFunc(pattern string, fn HandlerFunc[T])

	// Startup start all registered components
	Startup(ctx context.Context) (err error)

	// Shutdown shutdown all registered components
	Shutdown(ctx context.Context) (err error)
}

type componentRegistration struct {
	name     string
	startup  LifecycleFunc
	check    LifecycleFunc
	shutdown LifecycleFunc
}

type app[T Context] struct {
	// before init
	mu   sync.Locker
	cf   ContextFactory[T]
	opts options

	// after init
	regs []*componentRegistration
	init []*componentRegistration

	mux *http.ServeMux

	h http.Handler
	p http.Handler

	pprof http.Handler

	cc chan struct{}

	readinessFailed int64
}

func (a *app[T]) Component(name string, fns ...LifecycleFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, item := range a.regs {
		if item.name == name {
			panic("summer: duplicated component with name: " + name)
		}
	}
	reg := &componentRegistration{
		name: name,
	}
	if len(fns) > 0 {
		reg.startup = fns[0]
	}
	if len(fns) > 1 {
		reg.check = fns[1]
	}
	if len(fns) > 2 {
		reg.shutdown = fns[2]
	}
	a.regs = append(a.regs, reg)
}

func (a *app[T]) executeChecks(ctx context.Context) (res string, failed bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sb := &strings.Builder{}

	for _, item := range a.regs {
		if item.check == nil {
			return
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(item.name)
		sb.WriteString(": ")
		if err := item.check(ctx); err == nil {
			sb.WriteString("OK")
		} else {
			failed = true
			sb.WriteString(err.Error())
		}
	}

	res = sb.String()

	if res == "" {
		res = "OK"
	}
	return
}

func (a *app[T]) HandleFunc(pattern string, fn HandlerFunc[T]) {
	a.mux.Handle(
		pattern,
		otelhttp.WithRouteTag(
			pattern,
			http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				c := a.cf(rw, req)
				func() {
					defer c.Perform()
					fn(c)
				}()
			}),
		),
	)
}

func (a *app[T]) Startup(ctx context.Context) (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	defer func() {
		if err == nil {
			return
		}
		for _, item := range a.init {
			_ = item.shutdown(ctx)
		}
		a.init = nil
	}()

	for _, item := range a.regs {
		if item.startup != nil {
			if err = item.startup(ctx); err != nil {
				return
			}
		}
		a.init = append(a.init, item)
	}

	return
}

func (a *app[T]) Shutdown(ctx context.Context) (err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, item := range a.init {
		if err1 := item.shutdown(ctx); err1 != nil {
			if err == nil {
				err = err1
			} else {
				err = errors.New(err.Error() + "; " + err1.Error())
			}
		}
	}
	a.init = nil

	return
}

func (a *app[T]) initialize() {
	// promhttp handler
	a.p = promhttp.Handler()

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
		r, failed := a.executeChecks(req.Context())
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
		a.p.ServeHTTP(rw, req)
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
		mu: &sync.Mutex{},
		cf: cf,

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
