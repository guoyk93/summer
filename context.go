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

type Context interface {
	context.Context

	Req() *http.Request
	Res() http.ResponseWriter

	Bind(data interface{})

	Code(code int)
	Body(contentType string, buf []byte)
	Text(s string)
	JSON(data interface{})

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
	m, err := flattenRequest(c.req)
	if err != nil {
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

func (c *summerContext) Code(code int) {
	c.code = code
}
func (c *summerContext) Body(contentType string, buf []byte) {
	c.rw.Header().Set("Content-Type", contentType)
	c.rw.Header().Set("Content-Length", strconv.Itoa(len(buf)))
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
