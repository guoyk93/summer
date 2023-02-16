package summer

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApp(t *testing.T) {
	bad := true

	a := Basic(WithReadinessCascade(1), WithConcurrency(1))
	a.Component("test-1", nil, func(ctx context.Context) (err error) {
		if bad {
			return errors.New("test-failed")
		} else {
			return
		}
	})
	a.HandleFunc("/test", func(ctx Context) {
		ctx.Text("OK")
	})

	rw, req := httptest.NewRecorder(), httptest.NewRequest("GET", "https://exmaple.com/debug/alive", nil)
	a.ServeHTTP(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	require.Equal(t, "OK", rw.Body.String())

	rw, req = httptest.NewRecorder(), httptest.NewRequest("GET", "https://exmaple.com/debug/ready", nil)
	a.ServeHTTP(rw, req)

	require.Equal(t, http.StatusInternalServerError, rw.Code)
	require.Equal(t, "test-1: test-failed", rw.Body.String())

	rw, req = httptest.NewRecorder(), httptest.NewRequest("GET", "https://exmaple.com/debug/ready", nil)
	a.ServeHTTP(rw, req)

	require.Equal(t, http.StatusInternalServerError, rw.Code)
	require.Equal(t, "test-1: test-failed", rw.Body.String())

	rw, req = httptest.NewRecorder(), httptest.NewRequest("GET", "https://exmaple.com/debug/alive", nil)
	a.ServeHTTP(rw, req)

	require.Equal(t, http.StatusInternalServerError, rw.Code)
	require.Equal(t, "CASCADED", rw.Body.String())

	bad = false

	rw, req = httptest.NewRecorder(), httptest.NewRequest("GET", "https://exmaple.com/debug/ready", nil)
	a.ServeHTTP(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	require.Equal(t, "test-1: OK", rw.Body.String())

	rw, req = httptest.NewRecorder(), httptest.NewRequest("GET", "https://exmaple.com/debug/alive", nil)
	a.ServeHTTP(rw, req)

	require.Equal(t, http.StatusOK, rw.Code)
	require.Equal(t, "OK", rw.Body.String())

	rw, req = httptest.NewRecorder(), httptest.NewRequest("GET", "https://exmaple.com/test", nil)
	a.ServeHTTP(rw, req)

}

func TestAppLifecycle(t *testing.T) {
	a := Basic()
	var (
		t1a bool
		t1b bool
		t1c bool
		t2a bool
		t2b bool
		t2c bool
	)
	a.Component("test-1", func(ctx context.Context) (err error) {
		t1a = true
		return
	}, func(ctx context.Context) (err error) {
		t1b = true
		return
	}, func(ctx context.Context) (err error) {
		t1c = true
		return
	})
	a.Component("test-2", func(ctx context.Context) (err error) {
		t2a = true
		err = errors.New("BBB")
		return
	}, func(ctx context.Context) (err error) {
		t2b = true
		return
	}, func(ctx context.Context) (err error) {
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
