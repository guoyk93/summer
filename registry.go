package summer

import (
	"context"
	"errors"
	"sync"
)

// LifecycleFunc lifecycle function for component
type LifecycleFunc func(ctx context.Context) (err error)

type Registry interface {
	// Component register a component
	//
	// In order of `startup`, `check` and `shutdown`
	Component(name string) Registration

	// Startup start all registered components
	Startup(ctx context.Context) (err error)

	// Check run all checks
	Check(ctx context.Context, fn func(name string, err error))

	// Shutdown shutdown all registered components
	Shutdown(ctx context.Context) (err error)
}

// Registration a registration in [Registry]
type Registration interface {
	// Name returns name of registration
	Name() string

	// Startup set startup function
	Startup(fn LifecycleFunc) Registration

	// Check set check function
	Check(fn LifecycleFunc) Registration

	// Shutdown set shutdown function
	Shutdown(fn LifecycleFunc) Registration
}

type registration struct {
	name     string
	startup  LifecycleFunc
	check    LifecycleFunc
	shutdown LifecycleFunc
}

func (r *registration) Name() string {
	return r.name
}

func (r *registration) Startup(fn LifecycleFunc) Registration {
	r.startup = fn
	return r
}

func (r *registration) Check(fn LifecycleFunc) Registration {
	r.check = fn
	return r
}

func (r *registration) Shutdown(fn LifecycleFunc) Registration {
	r.shutdown = fn
	return r
}

type registry struct {
	mu   sync.Locker
	regs []*registration
	init []*registration
}

func (a *registry) Component(name string) Registration {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, item := range a.regs {
		if item.name == name {
			panic("duplicated component with name: " + name)
		}
	}

	reg := &registration{
		name: name,
	}

	a.regs = append(a.regs, reg)

	return reg
}

func (a *registry) Startup(ctx context.Context) (err error) {
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

func (a *registry) Check(ctx context.Context, fn func(name string, err error)) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, item := range a.regs {
		if item.check == nil {
			fn(item.name, nil)
		} else {
			fn(item.name, item.check(ctx))
		}
	}

	return
}

func (a *registry) Shutdown(ctx context.Context) (err error) {
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

func NewRegistry() Registry {
	return &registry{mu: &sync.Mutex{}}
}
