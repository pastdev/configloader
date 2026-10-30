//nolint:goconst // explicit strings have explanatory value in tests
package meta_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pastdev/configloader/pkg/meta"
	"github.com/stretchr/testify/require"
)

// write creates each of files (slash separated paths relative to root) with
// its content, and returns root.
func write(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		err := os.MkdirAll(filepath.Dir(path), 0o700)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(content), 0o600)
		require.NoError(t, err)
	}
	return root
}

func TestAddFuncs(t *testing.T) {
	tester := func(t *testing.T, name string, rel string, expected string) {
		t.Helper()
		root := write(t, map[string]string{
			"config.yml":        "",
			"keys/gitlab.token": "secret",
		})
		funcs := map[string]any{}
		meta.AddFuncs(funcs)

		factory, ok := funcs[name].(func(meta.LoadContext) any)
		require.True(t, ok, "expected %s to be a LoadContext factory", name)
		f, ok := factory(meta.LoadContext{File: filepath.Join(root, "config.yml")}).(func(string, ...string) (string, error))
		require.True(t, ok, "expected %s factory to return a template func", name)

		actual, err := f(rel)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(root, filepath.FromSlash(expected)), actual)
	}

	t.Run("relativeFile is bound to the load context", func(t *testing.T) {
		tester(t, "relativeFile", "keys/gitlab.token", "keys/gitlab.token")
	})

	t.Run("relativeDir is bound to the load context", func(t *testing.T) {
		tester(t, "relativeDir", "keys", "keys")
	})
}

func TestRelativeFile(t *testing.T) {
	// tester writes files under a temp root and resolves rel with a
	// LoadContext whose File and Dirs are relative to that root. expected is
	// relative to root.
	tester := func(
		t *testing.T,
		files map[string]string,
		file string,
		dirs []string,
		rel string,
		mode []string,
		expected string,
	) {
		t.Helper()
		root := write(t, files)
		actual, err := meta.RelativeFile(loadContext(root, file, dirs), rel, mode...)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(root, filepath.FromSlash(expected)), actual)
	}

	errorTester := func(
		t *testing.T,
		files map[string]string,
		file string,
		rel string,
		mode []string,
		contains string,
	) {
		t.Helper()
		root := write(t, files)
		_, err := meta.RelativeFile(loadContext(root, file, nil), rel, mode...)
		require.Error(t, err)
		require.ErrorContains(t, err, contains)
	}

	t.Run("this is the default mode", func(t *testing.T) {
		tester(t,
			map[string]string{"keys/gitlab.token": "secret"},
			"config.yml", nil,
			"keys/gitlab.token", nil,
			"keys/gitlab.token")
	})

	t.Run("this resolves against the dir of the current file", func(t *testing.T) {
		tester(t,
			map[string]string{"app/keys/gitlab.token": "secret"},
			"app/config.yml", nil,
			"keys/gitlab.token", []string{"this"},
			"app/keys/gitlab.token")
	})

	t.Run("first takes the earliest match", func(t *testing.T) {
		tester(t,
			map[string]string{
				"system/keys/gitlab.token": "system",
				"user/keys/gitlab.token":   "user",
			},
			"", []string{"system", "user"},
			"keys/gitlab.token", []string{"first"},
			"system/keys/gitlab.token")
	})

	t.Run("last takes the latest match", func(t *testing.T) {
		tester(t,
			map[string]string{
				"system/keys/gitlab.token": "system",
				"user/keys/gitlab.token":   "user",
			},
			"", []string{"system", "user"},
			"keys/gitlab.token", []string{"last"},
			"user/keys/gitlab.token")
	})

	t.Run("last falls back to an earlier match", func(t *testing.T) {
		tester(t,
			map[string]string{"system/keys/gitlab.token": "system"},
			"", []string{"system", "user"},
			"keys/gitlab.token", []string{"last"},
			"system/keys/gitlab.token")
	})

	t.Run("first skips a directory of the same name", func(t *testing.T) {
		tester(t,
			map[string]string{
				"system/keys/gitlab.token/.keep": "",
				"user/keys/gitlab.token":         "user",
			},
			"", []string{"system", "user"},
			"keys/gitlab.token", []string{"first"},
			"user/keys/gitlab.token")
	})

	t.Run("allows dots that are not a parent segment", func(t *testing.T) {
		tester(t,
			map[string]string{"foo..bar": "secret"},
			"config.yml", nil,
			"foo..bar", nil,
			"foo..bar")
	})

	t.Run("rejects a directory", func(t *testing.T) {
		errorTester(t,
			map[string]string{"keys/.keep": ""},
			"config.yml", "keys", nil,
			"not a file")
	})

	t.Run("rejects a missing match", func(t *testing.T) {
		errorTester(t,
			map[string]string{},
			"config.yml", "keys/gitlab.token", nil,
			"not found")
	})

	t.Run("rejects a missing match along dirs", func(t *testing.T) {
		errorTester(t,
			map[string]string{},
			"config.yml", "keys/gitlab.token", []string{"last"},
			"not found")
	})

	t.Run("rejects an unknown mode", func(t *testing.T) {
		errorTester(t,
			map[string]string{"keys/gitlab.token": "secret"},
			"config.yml", "keys/gitlab.token", []string{"other"},
			"unknown mode")
	})

	t.Run("rejects more than one mode", func(t *testing.T) {
		errorTester(t,
			map[string]string{"keys/gitlab.token": "secret"},
			"config.yml", "keys/gitlab.token", []string{"this", "last"},
			"at most one")
	})

	t.Run("rejects an empty path", func(t *testing.T) {
		errorTester(t,
			map[string]string{},
			"config.yml", "", nil,
			"empty")
	})

	t.Run("rejects an absolute path", func(t *testing.T) {
		errorTester(t,
			map[string]string{},
			"config.yml", "/etc/passwd", nil,
			"absolute")
	})

	t.Run("rejects a leading parent segment", func(t *testing.T) {
		errorTester(t,
			map[string]string{},
			"config.yml", "../secret", nil,
			"parent directory")
	})

	t.Run("rejects an inner parent segment", func(t *testing.T) {
		errorTester(t,
			map[string]string{},
			"config.yml", "foo/../bar", nil,
			"parent directory")
	})

	t.Run("this rejects no current file", func(t *testing.T) {
		errorTester(t,
			map[string]string{"keys/gitlab.token": "secret"},
			"", "keys/gitlab.token", nil,
			"no current file")
	})
}

func TestRelativeDir(t *testing.T) {
	tester := func(
		t *testing.T,
		files map[string]string,
		file string,
		dirs []string,
		rel string,
		mode []string,
		expected string,
	) {
		t.Helper()
		root := write(t, files)
		actual, err := meta.RelativeDir(loadContext(root, file, dirs), rel, mode...)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(root, filepath.FromSlash(expected)), actual)
	}

	errorTester := func(t *testing.T, files map[string]string, rel string, contains string) {
		t.Helper()
		root := write(t, files)
		_, err := meta.RelativeDir(loadContext(root, "config.yml", nil), rel)
		require.Error(t, err)
		require.ErrorContains(t, err, contains)
	}

	t.Run("this resolves against the dir of the current file", func(t *testing.T) {
		tester(t,
			map[string]string{"ts/.keep": ""},
			"config.yml", nil,
			"ts", nil,
			"ts")
	})

	t.Run("last takes the latest match", func(t *testing.T) {
		tester(t,
			map[string]string{
				"system/ts/.keep": "",
				"user/ts/.keep":   "",
			},
			"", []string{"system", "user"},
			"ts", []string{"last"},
			"user/ts")
	})

	t.Run("last skips a file of the same name", func(t *testing.T) {
		tester(t,
			map[string]string{
				"system/ts/.keep": "",
				"user/ts":         "not a dir",
			},
			"", []string{"system", "user"},
			"ts", []string{"last"},
			"system/ts")
	})

	t.Run("rejects a file", func(t *testing.T) {
		errorTester(t,
			map[string]string{"keys/gitlab.token": "secret"},
			"keys/gitlab.token",
			"not a directory")
	})
}

// loadContext returns a LoadContext with file and dirs made relative to root.
// An empty file stays empty.
func loadContext(root string, file string, dirs []string) meta.LoadContext {
	lc := meta.LoadContext{}
	if file != "" {
		lc.File = filepath.Join(root, filepath.FromSlash(file))
	}
	for _, dir := range dirs {
		lc.Dirs = append(lc.Dirs, filepath.Join(root, filepath.FromSlash(dir)))
	}
	return lc
}
