//nolint:goconst // explicit strings have explanatory value in tests
package config_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pastdev/configloader/pkg/config"
	"github.com/stretchr/testify/require"
)

type innerSource struct {
	config.SourceLoader
}

var _ config.WrappedSourceLoader = innerSource{}

func (s innerSource) InnerSource() config.SourceLoader {
	return s.SourceLoader
}

func TestLoadContext(t *testing.T) {
	unmarshal := config.YamlValueTemplateUnmarshal(nil)

	write := func(t *testing.T, root string, files map[string]string) {
		t.Helper()
		for name, content := range files {
			path := filepath.Join(root, filepath.FromSlash(name))
			err := os.MkdirAll(filepath.Dir(path), 0o700)
			require.NoError(t, err)
			err = os.WriteFile(path, []byte(content), 0o600)
			require.NoError(t, err)
		}
	}

	load := func(t *testing.T, sources ...config.SourceLoader) map[any]any {
		t.Helper()
		var actual map[any]any
		err := (config.Sources[map[any]any]{Sources: sources}).Load(&actual)
		require.NoError(t, err)
		return actual
	}

	loadErr := func(t *testing.T, sources ...config.SourceLoader) error {
		t.Helper()
		var actual map[any]any
		return (config.Sources[map[any]any]{Sources: sources}).Load(&actual)
	}

	t.Run("FileSource sets the current file", func(t *testing.T) {
		dir := t.TempDir()
		token := filepath.Join(dir, "keys", "gitlab.token")
		write(t, dir, map[string]string{
			"config.yml":        `token: '{{ relativeFile "keys/gitlab.token" }}'` + "\n",
			"keys/gitlab.token": "secret",
		})

		actual := load(t, config.FileSource{
			Path:      filepath.Join(dir, "config.yml"),
			Unmarshal: unmarshal,
		})
		require.Equal(t, token, actual["token"])
	})

	t.Run("DirSource sets the current file", func(t *testing.T) {
		dir := t.TempDir()
		tsDir := filepath.Join(dir, "ts")
		write(t, dir, map[string]string{
			"config.yml": `scripts: '{{ relativeDir "ts" "this" }}'` + "\n",
			"ts/.keep":   "",
		})

		actual := load(t, config.DirSource{
			Path:      dir,
			Unmarshal: unmarshal,
		})
		require.Equal(t, tsDir, actual["scripts"])
	})

	t.Run("dirs follow source order for first", func(t *testing.T) {
		root := t.TempDir()
		system := filepath.Join(root, "system")
		userDir := filepath.Join(root, "user")
		systemFile := filepath.Join(system, "ts", "filters", "audit.ts")
		write(t, root, map[string]string{
			"system/config.yml":          `filter: '{{ relativeFile "ts/filters/audit.ts" "first" }}'` + "\n",
			"system/ts/filters/audit.ts": "system",
			"user/ts/filters/audit.ts":   "user",
		})

		actual := load(t,
			config.FileSource{Path: filepath.Join(system, "config.yml"), Unmarshal: unmarshal},
			config.DirSource{Path: userDir},
		)
		require.Equal(t, systemFile, actual["filter"])
	})

	t.Run("last overlay wins even when later source has no yaml", func(t *testing.T) {
		root := t.TempDir()
		system := filepath.Join(root, "system")
		userDir := filepath.Join(root, "user")
		userFile := filepath.Join(userDir, "ts", "filters", "audit.ts")
		write(t, root, map[string]string{
			"system/config.yml":          `filter: '{{ relativeFile "ts/filters/audit.ts" "last" }}'` + "\n",
			"system/ts/filters/audit.ts": "system",
			"user/ts/filters/audit.ts":   "user",
		})

		actual := load(t,
			config.FileSource{Path: filepath.Join(system, "config.yml"), Unmarshal: unmarshal},
			config.DirSource{Path: userDir},
		)
		require.Equal(t, userFile, actual["filter"])
	})

	t.Run("last dir overlay", func(t *testing.T) {
		root := t.TempDir()
		system := filepath.Join(root, "system")
		userDir := filepath.Join(root, "user")
		userTS := filepath.Join(userDir, "ts")
		write(t, root, map[string]string{
			"system/config.yml": `scripts: '{{ relativeDir "ts" "last" }}'` + "\n",
			"system/ts/.keep":   "",
			"user/ts/.keep":     "",
		})

		actual := load(t,
			config.FileSource{Path: filepath.Join(system, "config.yml"), Unmarshal: unmarshal},
			config.DirSource{Path: userDir},
		)
		require.Equal(t, userTS, actual["scripts"])
	})

	t.Run("wrapped source dirs are searched", func(t *testing.T) {
		root := t.TempDir()
		system := filepath.Join(root, "system")
		userDir := filepath.Join(root, "user")
		userFile := filepath.Join(userDir, "keys", "gitlab.token")
		write(t, root, map[string]string{
			"system/config.yml":        `token: '{{ relativeFile "keys/gitlab.token" "last" }}'` + "\n",
			"system/keys/gitlab.token": "system",
			"user/keys/gitlab.token":   "user",
		})

		actual := load(t,
			config.FileSource{Path: filepath.Join(system, "config.yml"), Unmarshal: unmarshal},
			innerSource{SourceLoader: config.DirSource{Path: userDir}},
		)
		require.Equal(t, userFile, actual["token"])
	})

	t.Run("RawSource has no current file", func(t *testing.T) {
		err := loadErr(t, config.RawSource{
			Data:      []byte(`token: '{{ relativeFile "keys/gitlab.token" }}'` + "\n"),
			Unmarshal: unmarshal,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "no current file")
	})

	t.Run("concurrent loads are isolated", func(t *testing.T) {
		const n = 16
		dirs := make([]string, n)
		for i := range dirs {
			dirs[i] = t.TempDir()
			write(t, dirs[i], map[string]string{
				"config.yml":        `token: '{{ relativeFile "keys/gitlab.token" }}'` + "\n",
				"keys/gitlab.token": "secret",
			})
		}

		var wg sync.WaitGroup
		results := make([]any, n)
		errs := make([]error, n)
		for i := range dirs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var actual map[any]any
				errs[i] = (config.Sources[map[any]any]{
					Sources: []config.SourceLoader{
						config.FileSource{Path: filepath.Join(dirs[i], "config.yml"), Unmarshal: unmarshal},
					},
				}).Load(&actual)
				results[i] = actual["token"]
			}()
		}
		wg.Wait()

		for i := range dirs {
			require.NoError(t, errs[i])
			require.Equal(t, filepath.Join(dirs[i], "keys", "gitlab.token"), results[i])
		}
	})
}
