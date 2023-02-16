package summer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/guoyk93/rg"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Bind a generic version of [Context.Bind]
//
// example:
//
//		func actionValidate(c summer.Context) {
//			args := summer.Bind[struct {
//	       		Tenant       string `json:"header_x_tenant"`
//				Username     string `json:"username"`
//				Age 		 int    `json:"age,string"`
//			}](c)
//	        _ = args.Tenant
//	        _ = args.Username
//	        _ = args.Age
//		}
func Bind[T any](c Context) (o T) {
	c.Bind(&o)
	return
}

// Context the most basic context of a incoming request and corresponding response
type Context interface {
	// Context extend the [context.Context] interface by proxying to [http.Request.Context]
	context.Context

	// Req returns the underlying *http.Request
	Req() *http.Request
	// Res returns the underlying http.ResponseWriter
	Res() http.ResponseWriter

	// Bind unmarshal the request data into any struct with json tags
	//
	// HTTP header is prefixed with "header_"
	//
	// HTTP query is prefixed with "query_"
	//
	// both JSON and Form are supported
	Bind(data interface{})

	// Code set the response code, can be called multiple times
	Code(code int)

	// Body set the response body with content type, can be called multiple times
	Body(contentType string, buf []byte)

	// Text set the response body to plain text
	Text(s string)

	// JSON set the response body to json
	JSON(data interface{})

	// Perform actually perform the response
	// it is suggested to use in defer, recover() is included to recover from any panics
	Perform()
}

type basicContext struct {
	req *http.Request
	rw  http.ResponseWriter

	buf []byte

	code int
	body []byte

	recvOnce *sync.Once
	sendOnce *sync.Once
}

func (c *basicContext) Deadline() (deadline time.Time, ok bool) {
	return c.req.Context().Deadline()
}

func (c *basicContext) Done() <-chan struct{} {
	return c.req.Context().Done()
}

func (c *basicContext) Err() error {
	return c.req.Context().Err()
}

func (c *basicContext) Value(key any) any {
	return c.req.Context().Value(key)
}

func (c *basicContext) Req() *http.Request {
	return c.req
}

func (c *basicContext) Res() http.ResponseWriter {
	return c.rw
}

func (c *basicContext) receive() {
	var m = map[string]any{}
	if err := extractRequest(m, c.req); err != nil {
		Halt(err, HaltWithStatusCode(http.StatusBadRequest))
	}
	c.buf = rg.Must(json.Marshal(m))
}

func (c *basicContext) send() {
	c.rw.WriteHeader(c.code)
	_, _ = c.rw.Write(c.body)
}

func (c *basicContext) Bind(data interface{}) {
	c.recvOnce.Do(c.receive)
	rg.Must0(json.Unmarshal(c.buf, data))
}

func (c *basicContext) Code(code int) {
	c.code = code
}

func (c *basicContext) Body(contentType string, buf []byte) {
	c.rw.Header().Set("Content-Type", contentType)
	c.rw.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	c.rw.Header().Set("X-Content-Type-Options", "nosniff")
	c.body = buf
}

func (c *basicContext) Text(s string) {
	c.Body(ContentTypeTextPlainUTF8, []byte(s))
}

func (c *basicContext) JSON(data interface{}) {
	buf := rg.Must(json.Marshal(data))
	c.Body(ContentTypeApplicationJSONUTF8, buf)
}

func (c *basicContext) Perform() {
	if r := recover(); r != nil {
		var (
			e  error
			ok bool
		)
		if e, ok = r.(error); !ok {
			e = fmt.Errorf("panic: %v", r)
		}
		c.Code(StatusCodeFromError(e))
		c.JSON(BodyFromError(e))
	}
	c.sendOnce.Do(c.send)
}

// ContextFactory factory function for creating an extended [Context]
type ContextFactory[T Context] func(rw http.ResponseWriter, req *http.Request) T

// BasicContext context factory creating a basic [Context] implementation
func BasicContext(rw http.ResponseWriter, req *http.Request) Context {
	return &basicContext{
		req:      req,
		rw:       rw,
		code:     http.StatusOK,
		recvOnce: &sync.Once{},
		sendOnce: &sync.Once{},
	}
}

var (
	_ ContextFactory[Context] = BasicContext
)
