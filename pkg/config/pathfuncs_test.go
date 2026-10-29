//nolint:goconst // explicit strings have explanatory value in tests
package config_test

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/pastdev/configloader/pkg/config"
	"github.com/stretchr/testify/require"
)

type innerSource struct {
	config.SourceLoader
}

func (s innerSource) InnerSource() config.SourceLoader {
	return s.SourceLoader
}

func TestDefaultFuncMapPathHelpers(t *testing.T) {
	funcs := config.DefaultFuncMap()
	for _, name := range []string{"home", "user", "relativeFile", "relativeDir"} {
		_, ok := funcs[name]
		require.True(t, ok, "expected %s in DefaultFuncMap", name)
	}
}

func TestHomeAndUserTemplateFuncs(t *testing.T) {
	unmarshal := config.YamlValueTemplateUnmarshal(nil)

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	u, err := user.Current()
	require.NoError(t, err)

	var actual map[any]any
	err = unmarshal([]byte(`
home: '{{ home }}'
user: '{{ user }}'
`), &actual)
	require.NoError(t, err)
	require.Equal(t, map[any]any{
		"home": homeDir,
		"user": u.Username,
	}, actual)
}

func TestRelativePathFuncs(t *testing.T) {
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

	t.Run("this file default mode", func(t *testing.T) {
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

	t.Run("this dir from DirSource", func(t *testing.T) {
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

	t.Run("first wins", func(t *testing.T) {
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

	t.Run("relativeFile rejects a directory", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, map[string]string{
			"config.yml": `token: '{{ relativeFile "keys" }}'` + "\n",
			"keys/.keep": "",
		})

		err := loadErr(t, config.FileSource{
			Path:      filepath.Join(dir, "config.yml"),
			Unmarshal: unmarshal,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "not a file")
	})

	t.Run("relativeDir rejects a file", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, map[string]string{
			"config.yml":        `scripts: '{{ relativeDir "keys/gitlab.token" }}'` + "\n",
			"keys/gitlab.token": "secret",
		})

		err := loadErr(t, config.FileSource{
			Path:      filepath.Join(dir, "config.yml"),
			Unmarshal: unmarshal,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "not a directory")
	})

	t.Run("unknown mode", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, map[string]string{
			"config.yml":        `token: '{{ relativeFile "keys/gitlab.token" "other" }}'` + "\n",
			"keys/gitlab.token": "secret",
		})

		err := loadErr(t, config.FileSource{
			Path:      filepath.Join(dir, "config.yml"),
			Unmarshal: unmarshal,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "unknown mode")
	})

	t.Run("missing match", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, map[string]string{
			"config.yml": `token: '{{ relativeFile "keys/gitlab.token" }}'` + "\n",
		})

		err := loadErr(t, config.FileSource{
			Path:      filepath.Join(dir, "config.yml"),
			Unmarshal: unmarshal,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "not found")
	})

	t.Run("absolute path", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, map[string]string{
			"config.yml": `token: '{{ relativeFile "/etc/passwd" }}'` + "\n",
		})

		err := loadErr(t, config.FileSource{
			Path:      filepath.Join(dir, "config.yml"),
			Unmarshal: unmarshal,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "absolute")
	})

	t.Run("this without a current file", func(t *testing.T) {
		err := loadErr(t, config.RawSource{
			Data:      []byte(`token: '{{ relativeFile "keys/gitlab.token" }}'` + "\n"),
			Unmarshal: unmarshal,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "no current file")
	})

	t.Run("first skips a directory when a later file exists", func(t *testing.T) {
		root := t.TempDir()
		system := filepath.Join(root, "system")
		userDir := filepath.Join(root, "user")
		userFile := filepath.Join(userDir, "keys", "gitlab.token")
		write(t, root, map[string]string{
			"system/config.yml":              `token: '{{ relativeFile "keys/gitlab.token" "first" }}'` + "\n",
			"system/keys/gitlab.token/.keep": "",
			"user/keys/gitlab.token":         "user",
		})

		actual := load(t,
			config.FileSource{Path: filepath.Join(system, "config.yml"), Unmarshal: unmarshal},
			config.DirSource{Path: userDir},
		)
		require.Equal(t, userFile, actual["token"])
	})

	t.Run("dot dot rejected", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, map[string]string{
			"config.yml": `token: '{{ relativeFile "../secret" }}'` + "\n",
		})

		err := loadErr(t, config.FileSource{
			Path:      filepath.Join(dir, "config.yml"),
			Unmarshal: unmarshal,
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "parent directory")
	})
}
