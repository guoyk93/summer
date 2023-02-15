package summer

import (
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
)

type wrappedError struct {
	error
}

func (we wrappedError) Unwrap() error {
	return we.error
}

func TestErrorCode(t *testing.T) {

	ec := ErrorWithHTTPStatus(errors.New("TEST"), 400)
	require.Equal(t, "TEST", ec.Error())
	require.Equal(t, 400, HTTPStatusFromError(ec))
	require.Equal(t, 400, HTTPStatusFromError(wrappedError{error: ec}))
	require.Equal(t, 500, HTTPStatusFromError(errors.New("500")))
}

func TestErrorWithCode(t *testing.T) {
	ec := ErrorWithHTTPStatus(errors.New("TEST"), 400)
	require.Equal(t, "TEST", ec.Error())
}
