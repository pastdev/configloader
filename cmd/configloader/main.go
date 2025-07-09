package main

import (
	"fmt"
	"os"

	cobraconfig "github.com/pastdev/configloader/pkg/cobra"
	"github.com/pastdev/configloader/pkg/config"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func main() {
	cfg := cobraconfig.Config[map[any]any]{
		DefaultSources: config.Sources[map[any]any]{
			config.FileSource[map[any]any]{Path: "/etc/configloader.yml"},
			config.DirSource[map[any]any]{Path: "/etc/configloader.d"},
			config.FileSource[map[any]any]{Path: "~/.config/configloader.yml"},
			config.DirSource[map[any]any]{Path: "~/.config/configloader.d"},
		},
	}

	root := cobra.Command{
		Use:   "configloader",
		Short: `An example app for how to use configloader.`,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			// optionally set a logger for the config lib
			config.Logger = zerolog.New(os.Stderr).Level(zerolog.TraceLevel).With().Timestamp().Logger()
			// load the configuration
			err := cfg.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			return nil
		},
	}

	// use the config to add persistent flags to the root command so that they
	// are available to all subcommands
	cfg.PersistentFlags(&root).FileSourceVar(
		config.YamlUnmarshal,
		"config",
		"location of one or more config files")
	cfg.PersistentFlags(&root).DirSourceVar(
		config.YamlUnmarshal,
		"config-dir",
		"location of one or more config directories")

	// optionally add a `config` subcommand that allows viewing of the resulting
	// configuration
	cfg.AddSubCommandTo(&root)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
