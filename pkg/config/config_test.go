package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pastdev/configloader/pkg/config"
	"github.com/stretchr/testify/require"
)

type LoadTester[T any] struct {
	Files   map[string]string
	Sources config.Sources[T]
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

	src := config.Sources[T]{}
	for _, item := range loader.Sources {
		switch s := item.(type) {
		case config.DirSource[T]:
			var path string
			if strings.HasPrefix(s.Path, "~/") {
				path = s.Path
			} else {
				path = filepath.Join(testDir, s.Path)
			}
			src = append(src, config.DirSource[T]{Path: path, Unmarshal: s.Unmarshal})
		case config.FileSource[T]:
			var path string
			if strings.HasPrefix(s.Path, "~/") {
				path = s.Path
			} else {
				path = filepath.Join(testDir, s.Path)
			}
			src = append(src, config.FileSource[T]{Path: path, Unmarshal: s.Unmarshal})
		case config.RawSource[T]:
			src = append(src, s)
		}
	}

	err = src.Load(&actual)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestLoad(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		LoadTester[map[any]any]{}.Test(t, map[any]any{}, map[any]any{})
	})

	t.Run("simple memory", func(t *testing.T) {
		LoadTester[map[any]any]{
			Sources: config.Sources[map[any]any]{
				config.RawSource[map[any]any]{Data: []byte(`{"foo":"bar"}`)},
			},
		}.Test(t, map[any]any{"foo": "bar"}, map[any]any{})
	})

	t.Run("memory override", func(t *testing.T) {
		LoadTester[map[any]any]{
			Sources: config.Sources[map[any]any]{
				config.RawSource[map[any]any]{Data: []byte(`{"foo":"bar","hip":"hop"}`)},
				config.RawSource[map[any]any]{Data: []byte(`{"foo":"baz"}`)},
			},
		}.Test(t, map[any]any{"foo": "baz", "hip": "hop"}, map[any]any{})
	})

	t.Run("simple file", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{"config.yml": `{"foo":"bar"}`},
			Sources: config.Sources[map[any]any]{
				config.FileSource[map[any]any]{Path: "config.yml"},
			},
		}.Test(t, map[any]any{"foo": "bar"}, map[any]any{})
	})

	t.Run("file override", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{
				"config.yml":  `{"foo":"bar","hip":"hop"}`,
				"config2.yml": `{"foo":"baz"}`,
			},
			Sources: config.Sources[map[any]any]{
				config.FileSource[map[any]any]{Path: "config.yml"},
				config.FileSource[map[any]any]{Path: "config2.yml"},
			},
		}.Test(t, map[any]any{"foo": "baz", "hip": "hop"}, map[any]any{})
	})

	t.Run("homedir file", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{"~/config.yml": `{"foo":"bar"}`},
			Sources: config.Sources[map[any]any]{
				config.FileSource[map[any]any]{Path: "~/config.yml"},
			},
		}.Test(t, map[any]any{"foo": "bar"}, map[any]any{})
	})

	t.Run("missing file", func(t *testing.T) {
		LoadTester[map[any]any]{
			Sources: config.Sources[map[any]any]{
				config.FileSource[map[any]any]{Path: "config.yml"},
			},
		}.Test(t, map[any]any{}, map[any]any{})
	})

	t.Run("file incorrect name", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{
				"config.yml": `{"foo":"bar","hip":"hop"}`,
			},
			Sources: config.Sources[map[any]any]{
				config.FileSource[map[any]any]{Path: "incorrect_name_config.yml"},
			},
		}.Test(t, map[any]any{}, map[any]any{})
	})

	t.Run("missing homedir file", func(t *testing.T) {
		LoadTester[map[any]any]{
			Sources: config.Sources[map[any]any]{
				config.FileSource[map[any]any]{Path: "~/config.yml"},
			},
		}.Test(t, map[any]any{}, map[any]any{})
	})

	t.Run("simple dir", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{"app/config.yml": `{"foo":"bar"}`},
			Sources: config.Sources[map[any]any]{
				config.DirSource[map[any]any]{Path: "app"},
			},
		}.Test(t, map[any]any{"foo": "bar"}, map[any]any{})
	})

	t.Run("single dir override", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{
				"app/config.yml":  `{"foo":"bar","hip":"hop"}`,
				"app/config2.yml": `{"foo":"baz"}`,
			},
			Sources: config.Sources[map[any]any]{
				config.DirSource[map[any]any]{Path: "app"},
			},
		}.Test(t, map[any]any{"foo": "baz", "hip": "hop"}, map[any]any{})
	})

	t.Run("multiple dir override", func(t *testing.T) {
		LoadTester[map[any]any]{
			Files: map[string]string{
				"system/app/config.yml": `{"foo":"bar","hip":"hop"}`,
				"user/app/config.yml":   `{"foo":"baz"}`,
			},
			Sources: config.Sources[map[any]any]{
				config.DirSource[map[any]any]{Path: "system/app"},
				config.DirSource[map[any]any]{Path: "user/app"},
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
			Sources: config.Sources[cfg]{
				config.DirSource[cfg]{Path: "app"},
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
			Sources: config.Sources[cfg]{
				config.DirSource[cfg]{Path: "app"},
				config.FileSource[cfg]{Path: "other/config.yml"},
			},
		}.Test(t, cfg{Foo: "baz", Hip: "hop"}, actual)
	})
}
