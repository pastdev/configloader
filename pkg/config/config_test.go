//nolint:goconst // explicit strings have explanatory value in tests
package config_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pastdev/configloader/pkg/config"
	"github.com/stretchr/testify/require"
)

type LoadTester[T any] struct {
	Convert func(merged any, cfg *T) error
	Files   map[string]string
	Sources []config.SourceLoader
}

func (loader LoadTester[T]) Test(t *testing.T, expected T, actual T) {
	testDir := t.TempDir()
	homeDir := t.TempDir()

	// ensure os.UserHomeDir finds explicit variable instead of bleed through
	// to environment tests are run in
	homeEnv := "HOME"
	switch runtime.GOOS {
	case "windows":
		homeEnv = "USERPROFILE"
	case "plan9":
		homeEnv = "home"
	}
	currHome, ok := os.LookupEnv(homeEnv)
	err := os.Setenv(homeEnv, homeDir)
	require.NoError(t, err)
	defer func() {
		if ok {
			_ = os.Setenv(homeEnv, currHome)
		} else {
			_ = os.Unsetenv(homeEnv)
		}
	}()

	for file, content := range loader.Files {
		var path string
		if strings.HasPrefix(file, "~/") {
			path = filepath.Join(homeDir, strings.TrimPrefix(file, "~/"))
		} else {
			path = filepath.Join(testDir, file)
		}
		err := os.MkdirAll(filepath.Dir(path), 0700)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(content), 0600)
		require.NoError(t, err)
	}

	src := []config.SourceLoader{}
	for _, item := range loader.Sources {
		switch s := item.(type) {
		case config.DirSource:
			var path string
			if strings.HasPrefix(s.Path, "~/") {
				path = s.Path
			} else {
				path = filepath.Join(testDir, s.Path)
			}
			src = append(src, config.DirSource{Path: path, Unmarshal: s.Unmarshal})
		case config.FileSource:
			var path string
			if strings.HasPrefix(s.Path, "~/") {
				path = s.Path
			} else {
				path = filepath.Join(testDir, s.Path)
			}
			src = append(src, config.FileSource{Path: path, Unmarshal: s.Unmarshal})
		case config.RawSource:
			src = append(src, s)
		}
	}

	err = (config.Sources[T]{Sources: src, Convert: loader.Convert}).Load(&actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestLoad(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		LoadTester[map[any]any]{}.Test(t, map[any]any{}, map[any]any{})
	})

	t.Run("simple memory", func(t *testing.T) {
		LoadTester[map[any]any]{
			Sources: []config.SourceLoader{
				config.RawSource{Data: []byte(`{"foo":"bar"}`)},
			},
		}.Test(t, map[any]any{"foo": "bar"}, map[any]any{})
	})

	t.Run("memory override", func(t *testing.T) {
		LoadTester[map[any]any]{
			Sources: []config.SourceLoader{
				config.RawSource{Data: []byte(`{"foo":"bar","hip":"hop"}`)},
				config.RawSource{Data: []byte(`{"foo":"baz"}`)},
			},
		}.Test(t, map[any]any{"foo": "baz", "hip": "hop"}, map[any]any{})
	})

	t.Run("simple file", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{"config.yml": `{"foo":"bar"}`},
			Sources: []config.SourceLoader{
				config.FileSource{Path: "config.yml"},
			},
		}.Test(t, map[any]any{"foo": "bar"}, map[any]any{})
	})

	t.Run("file override", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{
				"config.yml":  `{"foo":"bar","hip":"hop"}`,
				"config2.yml": `{"foo":"baz"}`,
			},
			Sources: []config.SourceLoader{
				config.FileSource{Path: "config.yml"},
				config.FileSource{Path: "config2.yml"},
			},
		}.Test(t, map[any]any{"foo": "baz", "hip": "hop"}, map[any]any{})
	})

	t.Run("homedir file", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{"~/config.yml": `{"foo":"bar"}`},
			Sources: []config.SourceLoader{
				config.FileSource{Path: "~/config.yml"},
			},
		}.Test(t, map[any]any{"foo": "bar"}, map[any]any{})
	})

	t.Run("missing file", func(t *testing.T) {
		LoadTester[map[any]any]{
			Sources: []config.SourceLoader{
				config.FileSource{Path: "config.yml"},
			},
		}.Test(t, map[any]any{}, map[any]any{})
	})

	t.Run("file incorrect name", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{
				"config.yml": `{"foo":"bar","hip":"hop"}`,
			},
			Sources: []config.SourceLoader{
				config.FileSource{Path: "incorrect_name_config.yml"},
			},
		}.Test(t, map[any]any{}, map[any]any{})
	})

	t.Run("missing homedir file", func(t *testing.T) {
		LoadTester[map[any]any]{
			Sources: []config.SourceLoader{
				config.FileSource{Path: "~/config.yml"},
			},
		}.Test(t, map[any]any{}, map[any]any{})
	})

	t.Run("simple dir", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{"app/config.yml": `{"foo":"bar"}`},
			Sources: []config.SourceLoader{
				config.DirSource{Path: "app"},
			},
		}.Test(t, map[any]any{"foo": "bar"}, map[any]any{})
	})

	t.Run("single dir override", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{
				"app/config.yml":  `{"foo":"bar","hip":"hop"}`,
				"app/config2.yml": `{"foo":"baz"}`,
			},
			Sources: []config.SourceLoader{
				config.DirSource{Path: "app"},
			},
		}.Test(t, map[any]any{"foo": "baz", "hip": "hop"}, map[any]any{})
	})

	t.Run("multiple dir override", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{
				"system/app/config.yml": `{"foo":"bar","hip":"hop"}`,
				"user/app/config.yml":   `{"foo":"baz"}`,
			},
			Sources: []config.SourceLoader{
				config.DirSource{Path: "system/app"},
				config.DirSource{Path: "user/app"},
			},
		}.Test(t, map[any]any{"foo": "baz", "hip": "hop"}, map[any]any{})
	})

	t.Run("yaml tags", func(t *testing.T) {
		type cfg struct {
			Foo string `yaml:"not_foo"`
			Hip string `yaml:"not_hip"`
		}
		var actual cfg

		LoadTester[cfg]{
			Files: map[string]string{
				"app/config.yml":  `{"not_foo":"bar","not_hip":"hop"}`,
				"app/config2.yml": `{"not_foo":"baz"}`,
			},
			Sources: []config.SourceLoader{
				config.DirSource{Path: "app"},
			},
		}.Test(t, cfg{Foo: "baz", Hip: "hop"}, actual)
	})

	t.Run("mixed source with yaml tags", func(t *testing.T) {
		type cfg struct {
			Foo string `yaml:"not_foo"`
			Hip string `yaml:"not_hip"`
		}
		var actual cfg

		LoadTester[cfg]{
			Files: map[string]string{
				"app/config.yml":   `{"not_foo":"bar","not_hip":"hop"}`,
				"other/config.yml": `{"not_foo":"baz"}`,
			},
			Sources: []config.SourceLoader{
				config.DirSource{Path: "app"},
				config.FileSource{Path: "other/config.yml"},
			},
		}.Test(t, cfg{Foo: "baz", Hip: "hop"}, actual)
	})

	// Regression test: when overlaying config sources, updating a nested field
	// in a map[string]struct value should not replace the whole map entry.
	t.Run("nested map value struct override", func(t *testing.T) {
		type user struct {
			Name  string `yaml:"name"`
			Shell string `yaml:"shell"`
		}
		type app struct {
			User user `yaml:"user"`
		}
		type cfg struct {
			Apps map[string]app `yaml:"apps"`
		}

		var actual cfg

		LoadTester[cfg]{
			Sources: []config.SourceLoader{
				config.RawSource{
					Data: []byte(`---
apps:
  default:
    user:
      name: me
      shell: /bin/bash
`),
				},
				config.RawSource{
					Data: []byte(`---
apps:
  default:
    user:
      name: not_me
`),
				},
			},
		}.Test(t,
			cfg{
				Apps: map[string]app{
					"default": {
						User: user{
							Name:  "not_me",
							Shell: "/bin/bash",
						},
					},
				},
			},
			actual,
		)
	})

	t.Run("custom convert with json tags", func(t *testing.T) {
		type cfg struct {
			Foo string `json:"not_foo"`
			Hip string `json:"not_hip"`
		}
		var actual cfg

		LoadTester[cfg]{
			Convert: func(merged any, cfg *cfg) error {
				data, err := json.Marshal(merged)
				if err != nil {
					return fmt.Errorf("json marshal: %w", err)
				}
				return json.Unmarshal(data, cfg)
			},
			Sources: []config.SourceLoader{
				config.RawSource{
					Data:      []byte(`{"not_foo":"bar","not_hip":"hop"}`),
					Unmarshal: json.Unmarshal,
				},
				config.RawSource{
					Data:      []byte(`{"not_foo":"baz"}`),
					Unmarshal: json.Unmarshal,
				},
			},
		}.Test(t, cfg{Foo: "baz", Hip: "hop"}, actual)
	})

	t.Run("custom convert with nested map json tags", func(t *testing.T) {
		type user struct {
			Name  string `json:"name"`
			Shell string `json:"shell"`
		}
		type app struct {
			User user `json:"user"`
		}
		type cfg struct {
			Apps map[string]app `json:"apps"`
		}

		var actual cfg

		LoadTester[cfg]{
			Convert: func(merged any, cfg *cfg) error {
				data, err := json.Marshal(merged)
				if err != nil {
					return fmt.Errorf("json marshal: %w", err)
				}
				return json.Unmarshal(data, cfg)
			},
			Sources: []config.SourceLoader{
				config.RawSource{
					Data:      []byte(`{"apps":{"default":{"user":{"name":"me","shell":"/bin/bash"}}}}`),
					Unmarshal: json.Unmarshal,
				},
				config.RawSource{
					Data:      []byte(`{"apps":{"default":{"user":{"name":"not_me"}}}}`),
					Unmarshal: json.Unmarshal,
				},
			},
		}.Test(t,
			cfg{
				Apps: map[string]app{
					"default": {
						User: user{
							Name:  "not_me",
							Shell: "/bin/bash",
						},
					},
				},
			},
			actual,
		)
	})

	t.Run("non string yaml keys are stringified", func(t *testing.T) {
		LoadTester[map[string]any]{
			Sources: []config.SourceLoader{
				config.RawSource{
					Data: []byte(`---
1: one
true: two
`),
				},
			},
		}.Test(t,
			map[string]any{
				"1":    "one",
				"true": "two",
			},
			map[string]any{},
		)
	})
}
