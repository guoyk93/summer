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
	a.Component("test-1").Check(func(ctx context.Context) (err error) {
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
