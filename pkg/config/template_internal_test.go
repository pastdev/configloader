package config

import (
	"fmt"
	"os/user"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHomeAndUserFallbacks(t *testing.T) {
	oldHome := osUserHomeDir
	osUserHomeDir = func() (string, error) {
		return "", fmt.Errorf("boom")
	}
	t.Cleanup(func() { osUserHomeDir = oldHome })
	oldUser := userCurrent
	userCurrent = func() (*user.User, error) {
		return nil, fmt.Errorf("boom")
	}
	t.Cleanup(func() { userCurrent = oldUser })

	actual, err := home("/fallback/home")
	require.NoError(t, err)
	require.Equal(t, "/fallback/home", actual)

	actual, err = currentUsername("Fallback")
	require.NoError(t, err)
	require.Equal(t, "fallback", actual)

	actual, err = currentUsernameRaw("Fallback")
	require.NoError(t, err)
	require.Equal(t, "Fallback", actual)

	_, err = currentUsernameRaw("")
	require.Error(t, err)
}

func TestCurrentUsernameNormalization(t *testing.T) {
	old := userCurrent
	userCurrent = func() (*user.User, error) {
		return &user.User{Username: `CORP\Jane.Doe`}, nil
	}
	t.Cleanup(func() { userCurrent = old })

	actual, err := currentUsername()
	require.NoError(t, err)
	require.Equal(t, "jane.doe", actual)

	actual, err = currentUsernameRaw()
	require.NoError(t, err)
	require.Equal(t, `CORP\Jane.Doe`, actual)
}

func TestHomeAndUserLookupErrors(t *testing.T) {
	t.Run("home", func(t *testing.T) {
		old := osUserHomeDir
		osUserHomeDir = func() (string, error) {
			return "", fmt.Errorf("boom")
		}
		t.Cleanup(func() { osUserHomeDir = old })

		_, err := home()
		require.Error(t, err)
		require.ErrorContains(t, err, "home")
	})

	t.Run("user", func(t *testing.T) {
		old := userCurrent
		userCurrent = func() (*user.User, error) {
			return nil, fmt.Errorf("boom")
		}
		t.Cleanup(func() { userCurrent = old })

		_, err := currentUsername()
		require.Error(t, err)
		require.ErrorContains(t, err, "user")
	})
}
