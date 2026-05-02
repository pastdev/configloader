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
		if v, isSet := os.LookupEnv(name); isSet {
			err := os.Unsetenv(name)
			require.NoError(t, err)
			return func() { _ = os.Setenv(name, v) }
		}
		return func() {}
	}

	test := func(t *testing.T, funcName string, expected string) {
		funcmap := map[string]any{}
		AddFuncs(funcmap)

		f, ok := funcmap[funcName].(func() (string, error))
		require.True(t, ok)
		actual, err := f()
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

	t.Run("xdgCacheHome", func(t *testing.T) {
		defer unsetenv(t, "XDG_CACHE_HOME")()
		test(t, "xdgCacheHome", filepath.Join(homeDir, ".cache"))
	})

	t.Run("xdgCacheHome with env override", func(t *testing.T) {
		expected := "/some/cache"
		t.Setenv("XDG_CACHE_HOME", expected)
		test(t, "xdgCacheHome", expected)
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

	t.Run("xdgConfigHome", func(t *testing.T) {
		defer unsetenv(t, "XDG_CONFIG_HOME")()
		test(t, "xdgConfigHome", filepath.Join(homeDir, ".config"))
	})

	t.Run("xdgConfigHome with env override", func(t *testing.T) {
		expected := "/some/config/home"
		t.Setenv("XDG_CONFIG_HOME", expected)
		test(t, "xdgConfigHome", expected)
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

	t.Run("xdgDataHome", func(t *testing.T) {
		defer unsetenv(t, "XDG_DATA_HOME")()
		test(t, "xdgDataHome", filepath.Join(homeDir, ".local", "share"))
	})

	t.Run("xdgDataHome with env override", func(t *testing.T) {
		expected := "/some/data/home"
		t.Setenv("XDG_DATA_HOME", expected)
		test(t, "xdgDataHome", expected)
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

	t.Run("xdgRuntimeDir", func(t *testing.T) {
		defer unsetenv(t, "XDG_RUNTIME_DIR")()
		expected := fmt.Sprintf("/run/user/%s", u.Uid)
		test(t, "xdgRuntimeDir", expected)
	})

	t.Run("xdgRuntimeDir with env override", func(t *testing.T) {
		expected := "/some/runtime/dir"
		t.Setenv("XDG_RUNTIME_DIR", expected)
		test(t, "xdgRuntimeDir", expected)
	})
}
