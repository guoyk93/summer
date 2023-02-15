package summer

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContext(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/get?aaa=bbb", nil)
	rw := httptest.NewRecorder()
	ctx := newContext(rw, req)

	func() {
		defer ctx.Perform()

		type Request struct {
			AAA string `json:"aaa"`
		}
		var r Request

		ctx.Bind(&r)

		require.Equal(t, "bbb", r.AAA)

		ctx.Code(http.StatusTeapot)
		ctx.Text("OK")
	}()

	rw.Flush()

	require.Equal(t, http.StatusTeapot, rw.Code)
	require.Equal(t, "text/plain; charset=utf-8", rw.Header().Get("Content-Type"))
	require.Equal(t, "2", rw.Header().Get("Content-Length"))
	require.Equal(t, "OK", rw.Body.String())

}

func TestContextPanic(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/get?aaa=bbb", nil)
	rw := httptest.NewRecorder()
	ctx := newContext(rw, req)

	func() {
		defer ctx.Perform()

		type Request struct {
			AAA string `json:"aaa"`
		}
		var r Request

		ctx.Bind(&r)

		require.Equal(t, "bbb", r.AAA)

		panic("WWW")

		ctx.Code(http.StatusTeapot)
		ctx.Text("OK")
	}()

	rw.Flush()

	require.Equal(t, http.StatusInternalServerError, rw.Code)
	require.Equal(t, "application/json; charset=utf-8", rw.Header().Get("Content-Type"))
	require.Equal(t, `{"message":"panic: WWW"}`, rw.Body.String())
}
