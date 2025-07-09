package config

import (
	"fmt"
	"os"

	"github.com/pastdev/configloader/pkg/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Config provides utilities to integrate configuration loading into cobra
// CLI commands.
type Config[T any] struct {
	config T
	// DefaultSources are sources that you can configure in the code and allow
	// for the flags to replace at runtime.
	DefaultSources config.Sources[T]
	sources        config.Sources[T]
}

// Config returns the generated configuration object that will be loaded by the
// Load method.
func (c *Config[T]) Config() *T {
	return &c.config
}

// Load loads the configuration. If sources were set using the persistent flags,
// then the DefaultSources will be ignored. Otherwise, configurationis loaded
// from the DefaultSources.
func (c *Config[T]) Load() error {
	sources := c.sources
	if len(sources) == 0 {
		sources = c.DefaultSources
	}

	err := sources.Load(&c.config)
	if err != nil {
		return fmt.Errorf("configloader load sources: %w", err)
	}
	return nil
}

// PersistentFlags returns a factory for adding configuration source flags to
// the supplied root command.
func (c *Config[T]) PersistentFlags(root *cobra.Command) *flags[T] {
	return &flags[T]{
		config: c,
		root:   root,
	}
}

// AddSubCommandTo will add a config subcommand to the supplied root command.
// This subcommand will print out the configuration.
func (c *Config[T]) AddSubCommandTo(root *cobra.Command) {
	root.AddCommand(
		&cobra.Command{
			Use:   "config",
			Short: `Print out the config data.`,
			RunE: func(_ *cobra.Command, _ []string) error {
				err := yaml.NewEncoder(os.Stdout).Encode(c.Config())
				if err != nil {
					return fmt.Errorf("serialize config: %w", err)
				}
				return nil
			},
		})
}
