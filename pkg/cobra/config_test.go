package cobra

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pastdev/configloader/pkg/config"
	cobracmd "github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestPersistentFlags(t *testing.T) {
	type Cfg struct {
		Name    string `yaml:"name"`
		Port    int    `yaml:"port"`
		Enabled bool   `yaml:"enabled"`
	}

	unmarshal := func(b []byte, cfg *Cfg) error {
		return yaml.Unmarshal(b, cfg)
	}

	writeTestFS := func(t *testing.T, files map[string]string) string {
		t.Helper()

		root := t.TempDir()

		for name, contents := range files {
			fullPath := filepath.Join(root, filepath.FromSlash(name))

			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				t.Fatalf("mkdir %q: %v", filepath.Dir(fullPath), err)
			}

			if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
				t.Fatalf("write %q: %v", fullPath, err)
			}
		}

		return root
	}

	tester := func(
		t *testing.T,
		testFiles map[string]string,
		baseDefault bool,
		flags []string,
		expected Cfg,
	) {
		t.Helper()

		testDir := writeTestFS(t, testFiles)

		var defaultSource config.SourceLoader[Cfg]
		defaultSource = config.RawSource[Cfg]{
			Data: []byte(`
name: default
port: 8080
enabled: true
`),
		}
		if baseDefault {
			defaultSource = BaseSource(defaultSource)
		}

		loader := &ConfigLoader[Cfg]{DefaultSources: config.Sources[Cfg]{defaultSource}}

		var got Cfg

		root := &cobracmd.Command{
			Use:           "test",
			SilenceErrors: true,
			SilenceUsage:  true,
			RunE: func(_ *cobracmd.Command, _ []string) error {
				cfg, err := loader.Config()
				if err != nil {
					return err
				}

				got = *cfg
				return nil
			},
		}

		pf := loader.PersistentFlags(root)
		pf.FileSourceVarP(
			unmarshal,
			"config",
			"c",
			"location of one or more config files",
		)
		pf.DirSourceVarP(
			unmarshal,
			"config-dir",
			"d",
			"location of one or more config directories",
		)

		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		if err := os.Chdir(testDir); err != nil {
			t.Fatalf("chdir %q: %v", testDir, err)
		}
		t.Cleanup(func() {
			_ = os.Chdir(oldWD)
		})

		root.SetArgs(flags)

		if _, err := root.ExecuteC(); err != nil {
			t.Fatalf("execute %v: %v", flags, err)
		}

		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("got %+v, want %+v", got, expected)
		}
	}

	t.Run("no flags", func(t *testing.T) {
		tester(t, nil, false, []string{}, Cfg{
			Name:    "default",
			Port:    8080,
			Enabled: true,
		})
	})

	t.Run("config flag", func(t *testing.T) {
		tester(
			t,
			map[string]string{
				"file.yml": `
name: from-file
port: 9090
`,
			},
			false,
			[]string{"--config", "file.yml"},
			Cfg{
				Name: "from-file",
				Port: 9090,
			},
		)
	})

	t.Run("config-dir flag", func(t *testing.T) {
		tester(
			t,
			map[string]string{
				"moreConfig/00-name.yml": `
name: from-dir
`,
				"moreConfig/10-rest.yml": `
port: 1001
enabled: true
`,
			},
			false,
			[]string{"--config-dir", "moreConfig"},
			Cfg{
				Name:    "from-dir",
				Port:    1001,
				Enabled: true,
			},
		)
	})

	t.Run("complex", func(t *testing.T) {
		tester(
			t,
			map[string]string{
				"file.yml": `
name: from-file
port: 1111
`,
				"moreConfig/00-enabled.yml": `
enabled: true
`,
				"another_file.yml": `
name: final
port: 2222
`,
			},
			false,
			[]string{
				"--config", "file.yml",
				"--config-dir", "moreConfig",
				"--config", "another_file.yml",
			},
			Cfg{
				Name:    "final",
				Port:    2222,
				Enabled: true,
			},
		)
	})

	t.Run("config flag overlays base default source", func(t *testing.T) {
		tester(
			t,
			map[string]string{
				"file.yml": `
name: from-file
`,
			},
			true,
			[]string{"--config", "file.yml"},
			Cfg{
				Name:    "from-file",
				Port:    8080,
				Enabled: true,
			},
		)
	})
}
