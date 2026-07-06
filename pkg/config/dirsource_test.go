//nolint:goconst // explicit strings have explanatory value in tests
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pastdev/configloader/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDirSourceLoad(t *testing.T) {
	tester := func(t *testing.T, files map[string]string, expected map[any]any) {
		LoadTester[map[any]any]{
			Files: files,
			Sources: []config.SourceLoader{
				config.DirSource{Path: "app"},
			},
		}.Test(t, expected, map[any]any{})
	}

	errorTester := func(t *testing.T, files map[string]string) {
		dir := t.TempDir()
		for name, content := range files {
			err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600)
			require.NoError(t, err)
		}

		err := (config.DirSource{Path: dir}).Load(func(any) {})
		require.Error(t, err)
	}

	t.Run("default ignores non yaml files", func(t *testing.T) {
		tester(t,
			map[string]string{
				"app/config.yml": `{"foo":"bar"}`,
				// leading * is a yaml alias indicator, so this would fail
				// to parse if DirSource ever attempted to unmarshal it
				"app/.gitignore": "*.swp\n",
				"app/README.md":  "# not yaml\njust prose\n",
			},
			map[any]any{"foo": "bar"},
		)
	})

	t.Run("configloader widens include", func(t *testing.T) {
		tester(t,
			map[string]string{
				"app/config.yml": `{"foo":"bar","hip":"hop"}`,
				"app/zzz.json":   `{"foo":"baz"}`,
				"app/.configloader": `
include:
  - "*.yml"
  - "*.json"
`,
			},
			map[any]any{"foo": "baz", "hip": "hop"},
		)
	})

	t.Run("configloader narrows via exclude", func(t *testing.T) {
		tester(t,
			map[string]string{
				"app/config.yml":       `{"foo":"bar","hip":"hop"}`,
				"app/config.local.yml": `{"foo":"baz"}`,
				"app/.configloader": `
exclude:
  - "*.local.yml"
`,
			},
			map[any]any{"foo": "bar", "hip": "hop"},
		)
	})

	t.Run("include and exclude are and'd", func(t *testing.T) {
		tester(t,
			map[string]string{
				"app/config.yml":       `{"foo":"bar","hip":"hop"}`,
				"app/config.local.yml": `{"foo":"baz"}`,
				"app/notes.txt":        "not yaml, not eligible either way\n",
				"app/.configloader": `
include:
  - "*.yml"
exclude:
  - "*.local.yml"
`,
			},
			map[any]any{"foo": "bar", "hip": "hop"},
		)
	})

	t.Run("configloader excludes itself even when include matches everything", func(t *testing.T) {
		tester(t,
			map[string]string{
				"app/config.yml": `{"foo":"bar"}`,
				"app/notes.txt":  `{"other":"value"}`,
				"app/.configloader": `
include:
  - "*"
`,
			},
			map[any]any{"foo": "bar", "other": "value"},
		)
	})

	t.Run("malformed configloader is an error", func(t *testing.T) {
		errorTester(t, map[string]string{
			"config.yml":    `{"foo":"bar"}`,
			".configloader": "not: [valid",
		})
	})
}
