package xdg

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddFuncs(t *testing.T) {
	unsetenv := func(t *testing.T, name string) func() {
		t.Helper()
		if v, isSet := os.LookupEnv(name); isSet {
			err := os.Unsetenv(name)
			require.NoError(t, err)
			return func() { _ = os.Setenv(name, v) }
		}
		return func() {}
	}

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

	test := func(t *testing.T, funcName string, expected string, args ...string) {
		t.Helper()
		funcmap := map[string]any{}
		AddFuncs(funcmap)

		f, ok := funcmap[funcName].(func(...string) (string, error))
		require.True(t, ok)
		actual, err := f(args...)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}

	u, err := user.Current()
	require.NoError(t, err)
	homeDir := u.HomeDir

	t.Run("xdgBinHome", func(t *testing.T) {
		defer unsetenv(t, "XDG_BIN_HOME")()
		test(t, "xdgBinHome", filepath.Join(homeDir, ".local", "bin"))
	})

	t.Run("xdgBinHome with env override", func(t *testing.T) {
		expected := "/some/bin"
		t.Setenv("XDG_BIN_HOME", expected)
		test(t, "xdgBinHome", expected)
	})

	t.Run("xdgBinHome with empty env uses default", func(t *testing.T) {
		t.Setenv("XDG_BIN_HOME", "")
		test(t, "xdgBinHome", filepath.Join(homeDir, ".local", "bin"))
	})

	t.Run("xdgBinHome with fallback only when home unavailable", func(t *testing.T) {
		defer unsetenv(t, "XDG_BIN_HOME")()
		setUserHomeDir(t, func() (string, error) {
			return "", fmt.Errorf("boom")
		})
		test(t, "xdgBinHome", "/fallback/bin", "/fallback/bin")
	})

	t.Run("xdgCacheHome", func(t *testing.T) {
		defer unsetenv(t, "XDG_CACHE_HOME")()
		test(t, "xdgCacheHome", filepath.Join(homeDir, ".cache"))
	})

	t.Run("xdgCacheHome with env override", func(t *testing.T) {
		expected := "/some/cache"
		t.Setenv("XDG_CACHE_HOME", expected)
		test(t, "xdgCacheHome", expected)
	})

	t.Run("xdgCacheHome with empty env uses default", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		test(t, "xdgCacheHome", filepath.Join(homeDir, ".cache"))
	})

	t.Run("xdgCacheHome with fallback only when home unavailable", func(t *testing.T) {
		defer unsetenv(t, "XDG_CACHE_HOME")()
		setUserHomeDir(t, func() (string, error) {
			return "", fmt.Errorf("boom")
		})
		test(t, "xdgCacheHome", "/fallback/cache", "/fallback/cache")
	})

	t.Run("xdgConfigDirs", func(t *testing.T) {
		defer unsetenv(t, "XDG_CONFIG_DIRS")()
		test(t, "xdgConfigDirs", "/etc/xdg")
	})

	t.Run("xdgConfigDirs with env override", func(t *testing.T) {
		expected := "/some/config/dirs"
		t.Setenv("XDG_CONFIG_DIRS", expected)
		test(t, "xdgConfigDirs", expected)
	})

	t.Run("xdgConfigDirs with empty env uses default", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_DIRS", "")
		test(t, "xdgConfigDirs", "/etc/xdg")
	})

	t.Run("xdgConfigHome", func(t *testing.T) {
		defer unsetenv(t, "XDG_CONFIG_HOME")()
		test(t, "xdgConfigHome", filepath.Join(homeDir, ".config"))
	})

	t.Run("xdgConfigHome with env override", func(t *testing.T) {
		expected := "/some/config/home"
		t.Setenv("XDG_CONFIG_HOME", expected)
		test(t, "xdgConfigHome", expected)
	})

	t.Run("xdgConfigHome with empty env uses default", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		test(t, "xdgConfigHome", filepath.Join(homeDir, ".config"))
	})

	t.Run("xdgConfigHome with fallback only when home unavailable", func(t *testing.T) {
		defer unsetenv(t, "XDG_CONFIG_HOME")()
		setUserHomeDir(t, func() (string, error) {
			return "", fmt.Errorf("boom")
		})
		test(t, "xdgConfigHome", "/fallback/config/home", "/fallback/config/home")
	})

	t.Run("xdgDataDirs", func(t *testing.T) {
		defer unsetenv(t, "XDG_DATA_DIRS")()
		test(t, "xdgDataDirs", "/usr/local/share/:/usr/share/")
	})

	t.Run("xdgDataDirs with env override", func(t *testing.T) {
		expected := "/some/data/dirs"
		t.Setenv("XDG_DATA_DIRS", expected)
		test(t, "xdgDataDirs", expected)
	})

	t.Run("xdgDataDirs with empty env uses default", func(t *testing.T) {
		t.Setenv("XDG_DATA_DIRS", "")
		test(t, "xdgDataDirs", "/usr/local/share/:/usr/share/")
	})

	t.Run("xdgDataHome", func(t *testing.T) {
		defer unsetenv(t, "XDG_DATA_HOME")()
		test(t, "xdgDataHome", filepath.Join(homeDir, ".local", "share"))
	})

	t.Run("xdgDataHome with env override", func(t *testing.T) {
		expected := "/some/data/home"
		t.Setenv("XDG_DATA_HOME", expected)
		test(t, "xdgDataHome", expected)
	})

	t.Run("xdgDataHome with empty env uses default", func(t *testing.T) {
		t.Setenv("XDG_DATA_HOME", "")
		test(t, "xdgDataHome", filepath.Join(homeDir, ".local", "share"))
	})

	t.Run("xdgDataHome with fallback only when home unavailable", func(t *testing.T) {
		defer unsetenv(t, "XDG_DATA_HOME")()
		setUserHomeDir(t, func() (string, error) {
			return "", fmt.Errorf("boom")
		})
		test(t, "xdgDataHome", "/fallback/data/home", "/fallback/data/home")
	})

	t.Run("xdgStateHome", func(t *testing.T) {
		defer unsetenv(t, "XDG_STATE_HOME")()
		test(t, "xdgStateHome", filepath.Join(homeDir, ".local", "state"))
	})

	t.Run("xdgStateHome with env override", func(t *testing.T) {
		expected := "/some/state/home"
		t.Setenv("XDG_STATE_HOME", expected)
		test(t, "xdgStateHome", expected)
	})

	t.Run("xdgStateHome with empty env uses default", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "")
		test(t, "xdgStateHome", filepath.Join(homeDir, ".local", "state"))
	})

	t.Run("xdgStateHome with fallback only when home unavailable", func(t *testing.T) {
		defer unsetenv(t, "XDG_STATE_HOME")()
		setUserHomeDir(t, func() (string, error) {
			return "", fmt.Errorf("boom")
		})
		test(t, "xdgStateHome", "/fallback/state/home", "/fallback/state/home")
	})

	t.Run("xdgRuntimeDir", func(t *testing.T) {
		defer unsetenv(t, "XDG_RUNTIME_DIR")()
		test(t, "xdgRuntimeDir", fmt.Sprintf("/run/user/%s", u.Uid))
	})

	t.Run("xdgRuntimeDir with env override", func(t *testing.T) {
		expected := "/some/runtime/dir"
		t.Setenv("XDG_RUNTIME_DIR", expected)
		test(t, "xdgRuntimeDir", expected)
	})

	t.Run("xdgRuntimeDir with empty env uses replacement dir", func(t *testing.T) {
		t.Setenv("XDG_RUNTIME_DIR", "")
		test(t, "xdgRuntimeDir", fmt.Sprintf("/run/user/%s", u.Uid))
	})

	t.Run("xdgRuntimeDir with fallback only when replacement unavailable", func(t *testing.T) {
		defer unsetenv(t, "XDG_RUNTIME_DIR")()
		setUserCurrent(t, func() (*user.User, error) {
			return nil, fmt.Errorf("boom")
		})
		test(t, "xdgRuntimeDir", "/fallback/runtime/dir", "/fallback/runtime/dir")
	})
}
