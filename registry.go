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
	Component(name string, fns ...LifecycleFunc)

	// Startup start all registered components
	Startup(ctx context.Context) (err error)

	// Check run all checks
	Check(ctx context.Context, fn func(name string, err error))

	// Shutdown shutdown all registered components
	Shutdown(ctx context.Context) (err error)
}

type registration struct {
	name     string
	startup  LifecycleFunc
	check    LifecycleFunc
	shutdown LifecycleFunc
}

type registry struct {
	mu   sync.Locker
	regs []*registration
	init []*registration
}

func (a *registry) Component(name string, fns ...LifecycleFunc) {
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
