package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	cobraconfig "github.com/pastdev/configloader/pkg/cobra"
	"github.com/pastdev/configloader/pkg/config"
	"github.com/pastdev/configloader/pkg/log"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func fooCmd(cfgldr *cobraconfig.ConfigLoader[map[string]any]) *cobra.Command {
	return &cobra.Command{
		Use:   "foo",
		Short: `An example subcommand for how to use configloader to show the value of foo.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := cfgldr.Config()
			if err != nil {
				return fmt.Errorf("get config: %w", err)
			}

			fmt.Printf("foo is [%v]", (*cfg)["foo"])
			return nil
		},
	}
}

func main() {
	cfgldr := cobraconfig.ConfigLoader[map[string]any]{
		DefaultSources: config.Sources[map[string]any]{
			Sources: []config.SourceLoader{
				config.RawSource{
					Data: []byte(`foo: bar`),
				},
				config.FileSource{Path: "/etc/configloader.yml"},
				config.FileSource{
					Path: "/etc/configloader.tmpl.yml",
					Unmarshal: config.
						YamlValueTemplateUnmarshal(nil),
				},
				config.DirSource{Path: "/etc/configloader.d"},
				config.DirSource{
					Path: "/etc/configloader.tmpl.d",
					Unmarshal: config.
						YamlValueTemplateUnmarshal(nil),
				},
				config.FileSource{Path: "~/.config/configloader.yml"},
				config.FileSource{
					Path: "~/.config/configloader.tmpl.yml",
					Unmarshal: config.
						YamlValueTemplateUnmarshal(nil),
				},
				config.DirSource{Path: "~/.config/configloader.d"},
				config.DirSource{
					Path: "~/.config/configloader.tmpl.d",
					Unmarshal: config.
						YamlValueTemplateUnmarshal(nil),
				},
			},
		},
	}

	root := cobra.Command{
		Use:   "configloader",
		Short: `An example app for how to use configloader.`,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			// optionally set a logger for the config lib
			log.Logger = zerolog.New(os.Stderr).Level(zerolog.TraceLevel).With().Timestamp().Logger()
			return nil
		},
	}

	// use the config to add persistent flags to the root command so that they
	// are available to all subcommands
	cfgldr.PersistentFlags(&root).FileSourceVar(
		config.YamlUnmarshal(),
		"config",
		"location of one or more config files")
	cfgldr.PersistentFlags(&root).DirSourceVar(
		config.YamlUnmarshal(),
		"config-dir",
		"location of one or more config directories")

	// optionally add a `config` subcommand that allows viewing of the resulting
	// configuration
	cfgldr.AddSubCommandTo(
		&root,
		cobraconfig.WithConfigCommandOutput(
			"json",
			func(w io.Writer, cfg *map[string]any) error {
				err := json.NewEncoder(w).Encode(cfg)
				if err != nil {
					return fmt.Errorf("format json: %w", err)
				}
				return nil
			},
		),
		cobraconfig.WithConfigCommandSilenceUsage[map[string]any](true))

	// pass the config loader to subcommands so they can access .Config()
	root.AddCommand(fooCmd(&cfgldr))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
