package summer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/guoyk93/rg"
	"net/http"
	"strconv"
	"sync"
)

// Context context of a incoming request and corresponding response
type Context interface {
	context.Context

	// Req returns the underlying *http.Request
	Req() *http.Request
	// Res returns the underlying http.ResponseWriter
	Res() http.ResponseWriter

	// ClientIP client ip calculated from X-Forwarded-For header
	ClientIP() string

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

type summerContext struct {
	context.Context

	req *http.Request
	rw  http.ResponseWriter

	buf []byte

	code int
	body []byte

	recvOnce *sync.Once
	sendOnce *sync.Once
}

func (c *summerContext) Req() *http.Request {
	return c.req
}

func (c *summerContext) Res() http.ResponseWriter {
	return c.rw
}

func (c *summerContext) receive() {
	var m = map[string]any{}
	if err := flattenRequest(m, c.req); err != nil {
		panic(ErrorWithHTTPStatus(err, http.StatusBadRequest))
	}
	c.buf = rg.Must(json.Marshal(m))
}

func (c *summerContext) send() {
	c.rw.WriteHeader(c.code)
	_, _ = c.rw.Write(c.body)
}

func (c *summerContext) Bind(data interface{}) {
	c.recvOnce.Do(c.receive)
	rg.Must0(json.Unmarshal(c.buf, data))
}

func (c *summerContext) ClientIP() string {
	return extractClientIP(c.req)
}

func (c *summerContext) Code(code int) {
	c.code = code
}
func (c *summerContext) Body(contentType string, buf []byte) {
	c.rw.Header().Set("Content-Type", contentType)
	c.rw.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	c.rw.Header().Set("X-Content-Type-Options", "nosniff")
	c.body = buf
}

func (c *summerContext) Text(s string) {
	c.Body(ContentTypeTextPlainUTF8, []byte(s))
}

func (c *summerContext) JSON(data interface{}) {
	buf := rg.Must(json.Marshal(data))
	c.Body(ContentTypeApplicationJSONUTF8, buf)
}

func (c *summerContext) Perform() {
	if r := recover(); r != nil {
		var (
			message string
			code    = http.StatusInternalServerError
		)
		if re, ok := r.(error); ok {
			message = re.Error()
			code = HTTPStatusFromError(re)
		} else {
			message = fmt.Sprintf("panic: %v", r)
		}
		c.Code(code)
		c.JSON(map[string]any{"message": message})
	}
	c.sendOnce.Do(c.send)
}

func newContext(rw http.ResponseWriter, req *http.Request) *summerContext {
	return &summerContext{
		Context:  req.Context(),
		req:      req,
		rw:       rw,
		code:     http.StatusOK,
		recvOnce: &sync.Once{},
		sendOnce: &sync.Once{},
	}
}
