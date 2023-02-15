package summer

import (
	"net/http"
)

type errorWithHTTPStatus struct {
	error
	code int
}

func (e errorWithHTTPStatus) StatusCode() int {
	return e.code
}

func (e errorWithHTTPStatus) Unwrap() error {
	return e.error
}

// ErrorWithHTTPStatus inject a http status code into an error
func ErrorWithHTTPStatus(err error, code int) error {
	return errorWithHTTPStatus{
		error: err,
		code:  code,
	}
}

// HTTPStatusFromError extract http status code from an error
func HTTPStatusFromError(err error) (code int) {
	for {
		if ec, ok := err.(interface {
			StatusCode() int
		}); ok {
			return ec.StatusCode()
		}
		if eu, ok := err.(interface {
			Unwrap() error
		}); ok {
			err = eu.Unwrap()
		} else {
			break
		}
	}
	return http.StatusInternalServerError
}
