package run

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveMaxIterationsFlagTakesPrecedence(t *testing.T) {
	result := resolveMaxIterations(5, 2)
	require.Equal(t, 2, result)
}

func TestResolveMaxIterationsConfigDefaultUsedWhenFlagAbsent(t *testing.T) {
	result := resolveMaxIterations(7, 0)
	require.Equal(t, 7, result)
}
