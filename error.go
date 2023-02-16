package summer

import (
	"errors"
	"net/http"
)

const (
	HaltExtraKeyMessage = "message"
)

// HaltOption configuration function for [HaltError]
type HaltOption func(h *haltError)

// HaltWithStatusCode a [HaltOption] setting status code
func HaltWithStatusCode(code int) HaltOption {
	return func(h *haltError) {
		h.statusCode = code
	}
}

// HaltWithBadRequest alias to [HaltWithStatusCode] with [http.StatusBadRequest]
func HaltWithBadRequest() HaltOption {
	return HaltWithStatusCode(http.StatusBadRequest)
}

// HaltWithMessage a [HaltOption] overriding message key
func HaltWithMessage(m string) HaltOption {
	return HaltWithExtra(HaltExtraKeyMessage, m)
}

// HaltWithExtra a [HaltOption] setting extras with a key-value
func HaltWithExtra(k string, v any) HaltOption {
	return func(h *haltError) {
		if h.extras == nil {
			h.extras = map[string]any{}
		}
		h.extras[k] = v
	}
}

// HaltWithExtras a [HaltOption] setting extras with key-values
func HaltWithExtras(m map[string]any) HaltOption {
	return func(h *haltError) {
		if h.extras == nil {
			h.extras = map[string]any{}
		}
		for k, v := range m {
			h.extras[k] = v
		}
	}
}

type withStatusCode interface {
	StatusCode() int
}

type withExtract interface {
	ExtractExtras(m map[string]any)
}

type withUnwrap interface {
	Unwrap() error
}

var (
	_ withStatusCode = &haltError{}
	_ withExtract    = &haltError{}
	_ withUnwrap     = &haltError{}
)

type haltError struct {
	error
	statusCode int
	extras     map[string]any
}

func (h *haltError) Unwrap() error {
	return h.error
}

func (h *haltError) StatusCode() int {
	return h.statusCode
}

func (h *haltError) ExtractExtras(m map[string]any) {
	for k, v := range h.extras {
		m[k] = v
	}
}

// NewHaltError create a new [HaltError]
func NewHaltError(err error, opts ...HaltOption) error {
	he := &haltError{
		error:      err,
		statusCode: http.StatusInternalServerError,
	}
	for _, opt := range opts {
		opt(he)
	}
	return he
}

// Halt panic with [NewHaltError]
func Halt(err error, opts ...HaltOption) {
	panic(NewHaltError(err, opts...))
}

// HaltString panic with [NewHaltError] and [errors.New]
func HaltString(s string, opts ...HaltOption) {
	Halt(errors.New(s), opts...)
}

// StatusCodeFromError get status code from previous created [HaltError]
func StatusCodeFromError(err error) int {
	for {
		if err == nil {
			return http.StatusInternalServerError
		}
		if eh, ok := err.(withStatusCode); ok {
			return eh.StatusCode()
		}
		if eu, ok := err.(withUnwrap); ok {
			err = eu.Unwrap()
		} else {
			break
		}
	}
	return http.StatusInternalServerError
}

// BodyFromError extract extras from previous created [HaltError]
func BodyFromError(err error) (m map[string]any) {
	for {
		if err == nil {
			return
		}
		if m == nil {
			m = map[string]any{
				HaltExtraKeyMessage: err.Error(),
			}
		}
		if eh, ok := err.(withExtract); ok {
			eh.ExtractExtras(m)
		}
		if eu, ok := err.(withUnwrap); ok {
			err = eu.Unwrap()
		} else {
			break
		}
	}
	return
}
