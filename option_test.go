package summer

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOptions(t *testing.T) {
	opts := options{}
	WithConcurrency(2)(&opts)
	require.Equal(t, 2, opts.concurrency)

	opts = options{}
	WithReadinessCascade(2)(&opts)
	require.Equal(t, int64(2), opts.readinessCascade)

	opts = options{}
	WithLivenessPath("/aaa")(&opts)
	require.Equal(t, "/aaa", opts.livenessPath)

	opts = options{}
	WithReadinessPath("/aaa")(&opts)
	require.Equal(t, "/aaa", opts.readinessPath)

	opts = options{}
	WithMetricsPath("/aaa")(&opts)
	require.Equal(t, "/aaa", opts.metricsPath)
}
