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

func fooCmd(cfgldr *cobraconfig.ConfigLoader[map[any]any]) *cobra.Command {
	return &cobra.Command{
		Use:   "foo",
		Short: `An example subcommand for how to use configloader to show the value of foo.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := cfgldr.Config()
			if err != nil {
				return fmt.Errorf("get config: %w", err)
			}

			fmt.Printf("foo is [%s]", (*cfg)["foo"])
			return nil
		},
	}
}

func main() {
	cfgldr := cobraconfig.ConfigLoader[map[any]any]{
		DefaultSources: config.Sources[map[any]any]{
			config.FileSource[map[any]any]{Path: "/etc/configloader.yml"},
			config.FileSource[map[any]any]{
				Path: "/etc/configloader.tmpl.yml",
				Unmarshal: config.
					YamlValueTemplateUnmarshal[map[any]any](nil),
			},
			config.DirSource[map[any]any]{Path: "/etc/configloader.d"},
			config.DirSource[map[any]any]{
				Path: "/etc/configloader.tmpl.d",
				Unmarshal: config.
					YamlValueTemplateUnmarshal[map[any]any](nil),
			},
			config.FileSource[map[any]any]{Path: "~/.config/configloader.yml"},
			config.FileSource[map[any]any]{
				Path: "~/.config/configloader.tmpl.yml",
				Unmarshal: config.
					YamlValueTemplateUnmarshal[map[any]any](nil),
			},
			config.DirSource[map[any]any]{Path: "~/.config/configloader.d"},
			config.DirSource[map[any]any]{
				Path: "~/.config/configloader.tmpl.d",
				Unmarshal: config.
					YamlValueTemplateUnmarshal[map[any]any](nil),
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
		config.YamlUnmarshal[map[any]any](),
		"config",
		"location of one or more config files")
	cfgldr.PersistentFlags(&root).DirSourceVar(
		config.YamlUnmarshal[map[any]any](),
		"config-dir",
		"location of one or more config directories")

	// optionally add a `config` subcommand that allows viewing of the resulting
	// configuration
	cfgldr.AddSubCommandTo(
		&root,
		cobraconfig.WithConfigCommandOutput(
			"json",
			func(w io.Writer, cfg *map[any]any) error {
				jsonmap := map[string]any{}
				for k, v := range *cfg {
					jsonmap[fmt.Sprintf("%s", k)] = v
				}

				err := json.NewEncoder(w).Encode(jsonmap)
				if err != nil {
					return fmt.Errorf("format json: %w", err)
				}
				return nil
			},
		),
		cobraconfig.WithConfigCommandSilenceUsage[map[any]any](true))

	// pass the config loader to subcommands so they can access .Config()
	root.AddCommand(fooCmd(&cfgldr))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
