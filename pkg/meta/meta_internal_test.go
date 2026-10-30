package meta

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateRelativePath(t *testing.T) {
	require.Error(t, validateRelativePath(""))
	require.Error(t, validateRelativePath("/abs"))
	require.Error(t, validateRelativePath("../escape"))
	require.Error(t, validateRelativePath("foo/../bar"))
	require.NoError(t, validateRelativePath("keys/gitlab.token"))
	require.NoError(t, validateRelativePath("foo..bar"))
}

func TestParseRelativeMode(t *testing.T) {
	mode, err := parseRelativeMode(nil)
	require.NoError(t, err)
	require.Equal(t, relativeModeThis, mode)

	mode, err = parseRelativeMode([]string{relativeModeLast})
	require.NoError(t, err)
	require.Equal(t, relativeModeLast, mode)

	_, err = parseRelativeMode([]string{"other"})
	require.Error(t, err)

	_, err = parseRelativeMode([]string{relativeModeThis, relativeModeLast})
	require.Error(t, err)
}
