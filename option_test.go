package summer

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWithConcurrency(t *testing.T) {
	opts := options{}
	WithConcurrency(2)(&opts)
	require.Equal(t, 2, opts.concurrency)
}

func TestWithReadinessCascade(t *testing.T) {
	opts := options{}
	WithReadinessCascade(2)(&opts)
	require.Equal(t, int64(2), opts.readinessCascade)
}
