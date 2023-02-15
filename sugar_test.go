package summer

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBind(t *testing.T) {
	var hello string

	a := New()
	a.HandleFunc("/test", func(c Context) {
		args := Bind[struct {
			Hello string `json:"query_hello"`
		}](c)

		hello = args.Hello
	})

	req := httptest.NewRequest("GET", "https://example.com/test?hello=world", nil)

	a.ServeHTTP(httptest.NewRecorder(), req)

	require.Equal(t, "world", hello)
}

func TestPanic(t *testing.T) {
	var r any
	func() {
		defer func() {
			r = recover()
		}()
		Panic("hello", http.StatusTeapot)
	}()
	err, ok := r.(error)
	require.True(t, ok)
	require.Equal(t, "hello", err.Error())
	require.Equal(t, http.StatusTeapot, HTTPStatusFromError(err))
}
