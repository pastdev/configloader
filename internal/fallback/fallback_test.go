package fallback_test

import (
	"testing"

	"github.com/pastdev/configloader/internal/fallback"
	"github.com/stretchr/testify/require"
)

func TestFirst(t *testing.T) {
	_, ok := fallback.First()
	require.False(t, ok)

	_, ok = fallback.First("")
	require.False(t, ok)

	v, ok := fallback.First("a", "b")
	require.True(t, ok)
	require.Equal(t, "a", v)
}
