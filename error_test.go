package summer

import (
	"errors"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestHalt(t *testing.T) {
	var err error
	func() {
		defer func() {
			err = recover().(error)
		}()

		HaltString(
			"test",
			HaltWithStatusCode(http.StatusTeapot),
			HaltWithExtra("aaa", "bbb"),
			HaltWithExtras(map[string]any{
				"ccc": "ddd",
				"eee": "fff",
			}),
		)
	}()
	m := BodyFromError(err)
	require.Equal(t, http.StatusTeapot, StatusCodeFromError(err))
	require.Equal(t, map[string]any{"message": "test", "aaa": "bbb", "ccc": "ddd", "eee": "fff"}, m)

	func() {
		defer func() {
			err = recover().(error)
		}()

		HaltString(
			"test",
			HaltWithStatusCode(http.StatusTeapot),
			HaltWithExtras(map[string]any{
				"ccc": "ddd",
				"eee": "fff",
			}),
			HaltWithExtra("aaa", "bbb"),
			HaltWithMessage("test2"),
		)
	}()
	m = BodyFromError(err)
	require.Equal(t, http.StatusTeapot, StatusCodeFromError(err))
	require.Equal(t, map[string]any{"message": "test2", "aaa": "bbb", "ccc": "ddd", "eee": "fff"}, m)
}

func TesPanicError(t *testing.T) {
	var err error
	func() {
		defer func() {
			err = recover().(error)
		}()
		panic(errors.New("TEST1"))
	}()
	m := BodyFromError(err)
	require.Equal(t, http.StatusInternalServerError, StatusCodeFromError(err))
	require.Equal(t, map[string]any{"message": "TEST1"}, m)
}

func TesPanicAny(t *testing.T) {
	var err error
	func() {
		defer func() {
			err = recover().(error)
		}()
		panic("TEST1")
	}()
	m := BodyFromError(err)
	require.Equal(t, http.StatusInternalServerError, StatusCodeFromError(err))
	require.Equal(t, map[string]any{"message": "panic: TEST1"}, m)
}
