package summer

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRegistry(t *testing.T) {
	a := NewRegistry()
	var (
		t1a bool
		t1b bool
		t1c bool
		t2a bool
		t2b bool
		t2c bool
	)
	a.Component("test-1").
		Startup(func(ctx context.Context) (err error) {
			t1a = true
			return
		}).
		Check(func(ctx context.Context) (err error) {
			t1b = true
			return
		}).
		Shutdown(func(ctx context.Context) (err error) {
			t1c = true
			return
		})

	a.Component("test-2").
		Startup(func(ctx context.Context) (err error) {
			t2a = true
			return
		}).
		Check(func(ctx context.Context) (err error) {
			t2b = true
			return
		}).
		Shutdown(func(ctx context.Context) (err error) {
			t2c = true
			return
		})

	err := a.Startup(context.Background())

	require.NoError(t, err)
	require.True(t, t1a)
	require.False(t, t1b)
	require.False(t, t1c)
	require.True(t, t2a)
	require.False(t, t2b)
	require.False(t, t2c)

	a.Check(context.Background(), func(name string, err error) {})

	require.True(t, t1a)
	require.True(t, t1b)
	require.False(t, t1c)
	require.True(t, t2a)
	require.True(t, t2b)
	require.False(t, t2c)

	err = a.Shutdown(context.Background())

	require.NoError(t, err)
	require.True(t, t1a)
	require.True(t, t1b)
	require.True(t, t1c)
	require.True(t, t2a)
	require.True(t, t2b)
	require.True(t, t2c)
}

func TestRegistryFailed(t *testing.T) {
	a := NewRegistry()
	var (
		t1a bool
		t1b bool
		t1c bool
		t2a bool
		t2b bool
		t2c bool
	)
	a.Component("test-1").
		Startup(func(ctx context.Context) (err error) {
			t1a = true
			return
		}).
		Check(func(ctx context.Context) (err error) {
			t1b = true
			return
		}).
		Shutdown(func(ctx context.Context) (err error) {
			t1c = true
			return
		})

	a.Component("test-2").
		Startup(func(ctx context.Context) (err error) {
			t2a = true
			return errors.New("BBB")
		}).
		Check(func(ctx context.Context) (err error) {
			t2b = true
			return
		}).
		Shutdown(func(ctx context.Context) (err error) {
			t2c = true
			return
		})

	err := a.Startup(context.Background())

	require.Error(t, err)
	require.Equal(t, "BBB", err.Error())
	require.True(t, t1a)
	require.False(t, t1b)
	require.True(t, t1c)
	require.True(t, t2a)
	require.False(t, t2b)
	require.False(t, t2c)
}
