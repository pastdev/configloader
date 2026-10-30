package config

import (
	"fmt"
	"os"
	"os/user"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddBaseFuncs(t *testing.T) {
	setUserHomeDir := func(t *testing.T, f func() (string, error)) {
		t.Helper()
		old := osUserHomeDir
		osUserHomeDir = f
		t.Cleanup(func() { osUserHomeDir = old })
	}

	setUserCurrent := func(t *testing.T, f func() (*user.User, error)) {
		t.Helper()
		old := userCurrent
		userCurrent = f
		t.Cleanup(func() { userCurrent = old })
	}

	failLookups := func(t *testing.T) {
		t.Helper()
		setUserHomeDir(t, func() (string, error) { return "", fmt.Errorf("boom") })
		setUserCurrent(t, func() (*user.User, error) { return nil, fmt.Errorf("boom") })
	}

	call := func(t *testing.T, funcName string, args ...string) (string, error) {
		t.Helper()
		funcs := map[string]any{}
		AddBaseFuncs(funcs)

		f, ok := funcs[funcName].(func(...string) (string, error))
		require.True(t, ok)
		return f(args...)
	}

	tester := func(t *testing.T, funcName string, expected string, args ...string) {
		t.Helper()
		actual, err := call(t, funcName, args...)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}

	errorTester := func(t *testing.T, funcName string, args ...string) {
		t.Helper()
		_, err := call(t, funcName, args...)
		require.Error(t, err)
		require.ErrorContains(t, err, "boom")
	}

	t.Run("home", func(t *testing.T) {
		expected, err := os.UserHomeDir()
		require.NoError(t, err)
		tester(t, "home", expected)
	})

	t.Run("home with fallback", func(t *testing.T) {
		failLookups(t)
		tester(t, "home", "/fallback/home", "/fallback/home")
	})

	t.Run("home without fallback errors", func(t *testing.T) {
		failLookups(t)
		errorTester(t, "home")
	})

	t.Run("user", func(t *testing.T) {
		u, err := user.Current()
		require.NoError(t, err)
		expected := strings.ToLower(u.Username)
		if idx := strings.Index(expected, `\`); idx != -1 {
			expected = expected[idx+1:]
		}
		tester(t, "user", expected)
	})

	t.Run("user is lowercased without domain", func(t *testing.T) {
		setUserCurrent(t, func() (*user.User, error) {
			return &user.User{Username: `CORP\Jane.Doe`}, nil
		})
		tester(t, "user", "jane.doe")
	})

	t.Run("user with fallback is normalized", func(t *testing.T) {
		failLookups(t)
		tester(t, "user", "fallback", "Fallback")
	})

	t.Run("user without fallback errors", func(t *testing.T) {
		failLookups(t)
		errorTester(t, "user")
	})

	t.Run("userRaw is not normalized", func(t *testing.T) {
		setUserCurrent(t, func() (*user.User, error) {
			return &user.User{Username: `CORP\Jane.Doe`}, nil
		})
		tester(t, "userRaw", `CORP\Jane.Doe`)
	})

	t.Run("userRaw with fallback", func(t *testing.T) {
		failLookups(t)
		tester(t, "userRaw", "Fallback", "Fallback")
	})

	t.Run("userRaw with empty fallback errors", func(t *testing.T) {
		failLookups(t)
		errorTester(t, "userRaw", "")
	})
}
